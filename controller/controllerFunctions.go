// Package controller provides functions for handling HTTP requests and implementing business logic between the database and application.
package controller

import (
	"WIG-Server/components"
	"WIG-Server/db"
	"WIG-Server/messages"
	"WIG-Server/models"
	"WIG-Server/structs"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
returnError returns the given error code, a 'false' success status and message through fiber to the application.

@param c The fiber context containing the HTTP request and response objects.
@param code The error code to return via fiber
@param message The error message to return via fiber
@return error - An error, if any, that occurred during the process.
*/
func returnError(c *fiber.Ctx, code int, message string) error {
	return c.Status(code).JSON(fiber.Map{
		"success":false,
		"message":message})
}

/*
returnSuccess returns a 200 success code, a 'true' success status and a message through fiber to the application.

@param c The fiber context containing the HTTP request and response objects.
@param message The success message to return via fiber.
@return error - An error, if any, that occurred during the process.
*/
func returnSuccess (c *fiber.Ctx, message string) error {
	return c.Status(200).JSON(fiber.Map{
		"success":true,
		"message":message})
}

/*
validateToken checks if a users UID and token match and are valid.

@param c *fiber.Ctx - The fier context containing the HTTP request and response objects.
@param UID The users UID
@param token The users authentication token
@return error - An error that occured during the process or if the token does not match
*/
func validateToken(c *fiber.Ctx, uid string, token string) (int, error){
	// Check if UID and token exist
        if uid == "" {
                return 400, errors.New(messages.UIDEmpty)
        }

        if token == "" {
                return 400, errors.New(messages.TokenEmpty)
        }

        // Query for UID
        var user models.User
        result := db.DB.Where("user_uid = ?", uid).First(&user)

        // Check if UID was found
        if result.Error == gorm.ErrRecordNotFound {
                return 404, errors.New("UID " + messages.RecordNotFound)

        } else if result.Error != nil {
                return 400, errors.New(messages.ErrorWithConnection)
	}

        // Validate token
        if !components.ValidateToken(user.Username, user.Hash, token) {
                return 400, errors.New(messages.ErrorToken)
                }
	
	return 200, nil
}

/*
getOwnershipReponse takes an ownership struct and sets up the ownership response.

@param ownership The Ownership to convert to an ownership response
@return structs.OwnershipResponse The converted ownership response
*/
func getOwnershipReponse(ownership models.Ownership) structs.OwnershipResponse {
	
	var location models.Location
	result := db.DB.Where("location_uid = ?", ownership.ItemLocation).Find(&location)
	var locationName string

	if result.Error == gorm.ErrRecordNotFound {
        	locationName = "Not found" // TODO messages
	} else {
		locationName = location.LocationName
	}


	return structs.OwnershipResponse{
                        OwnershipUID: ownership.OwnershipUID,                                    
                        CustomItemName: ownership.CustomItemName,
                        CustItemImg: ownership.CustItemImg,
                        OwnedCustDesc: ownership.OwnedCustDesc,
                        ItemLocation: locationName,
                        ItemQR: ownership.ItemQR,
                        ItemTags: ownership.ItemTags,
                        ItemQuantity: ownership.ItemQuantity,
                        ItemCheckedOut: ownership.ItemCheckedOut,
                        ItemBorrower: ownership.ItemBorrower,} 	
}

/*
CheckQR takes a QR code as parameter, and checks whether it is an item, location or a unused QR.

@param c *fiber.Ctx - The fier context containing the HTTP request and response objects.
@return error - An error that occured during the process or if the token does not match
*/
func ScanCheckQR(c *fiber.Ctx) error {
	// Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
        if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}

	// Initialize variables
        uid := data["uid"]
        qr := c.Query("qr")

	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Check for empty fields
	if qr == "" {return returnError(c, 400, messages.QRMissing)}
  
        // Check if qr exists as location
        var location models.Location
        result := db.DB.Where("location_qr = ? AND location_owner = ?", qr, uid).First(&location)
	if location.LocationUID != 0 {return returnSuccess(c, messages.Location)
	} else if result.Error != nil && result.Error != gorm.ErrRecordNotFound {return returnError(c, 400, messages.ErrorWithConnection)}

	// Check if qr exists as ownership
	var ownership models.Ownership
	result = db.DB.Where("item_qr = ? AND item_owner = ?", qr, uid).First(&ownership)
	if ownership.OwnershipUID != 0 {return returnSuccess(c, messages.Ownership)}
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {return returnError(c, 400, messages.ErrorWithConnection)}

	return returnSuccess(c, messages.New)
}

/*
RecordExists checks a gorm.DB error message to see if a record existed in the database.

@param field A string representing the field that is getting checked.
@param result The gorm.DB result that may hold the error message.
@return int The HTTP error code to return
@return The error message to return
*/
func recordExists(field string, result *gorm.DB) (int, error) {

	if result.Error == gorm.ErrRecordNotFound {return 404, errors.New(field + messages.DoesNotExist)}
	if result.Error != nil {return 400, errors.New(result.Error.Error())}
	
	return 200, nil
}

/*
RecordNotInUse checks a gorm.DB error message to see if a record is in use in the database.

@param field A string representing the field that is getting checked.
@param result The gorm.DB result that may hold the error message.
@return int The HTTP error code to return
@return The error message to return
*/
func recordNotInUse(field string, result *gorm.DB) (int, error) {	
	
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {return 400, errors.New(result.Error.Error())}
 	if result.RowsAffected != 0 {return 400, errors.New(field + messages.RecordInUse)}
	
	return 200, nil
}

func createOwnership(uid string, barcode string) (models.Ownership, error){
	// Convert uid to int
	uidInt, err := strconv.Atoi(uid)
	if err != nil {return models.Ownership{}, errors.New(messages.ConversionError)}
	
	ownership := models.Ownership{
               	ItemOwner:uint(uidInt),
		ItemBarcode:barcode,
   	}
		
	db.DB.Create(&ownership)

	return ownership, nil
}
