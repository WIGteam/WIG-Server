/* Package controller provides functions for handling HTTP requests and implementing business logic between the database and application.
 */
package controller

import (
	"WIG-Server/db"
	"WIG-Server/messages"
	"WIG-Server/models"
	"WIG-Server/structs"
	"WIG-Server/upcitemdb"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
GetBarcode handles the functionality of returning any ownerships and items back after scanning a barcode.

@param c *fiber.Ctx
*/
func GetBarcode(c *fiber.Ctx) error {
	// Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
        if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}

	// Initialize variables
	uid := data["uid"]
	barcode := c.Query("barcode")
	
	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Validate barcode
	if barcode == "" {return returnError(c, 400, "Barcode required")} // TODO add message
	barcodeCheck, err := strconv.Atoi(barcode)
	if err != nil || barcodeCheck < 0 {return returnError(c, 400, "Barcode must be of int value")}

	// Check if item exists in local database
	var item models.Item
        result := db.DB.Where("barcode = ?", barcode).First(&item) 

        // If item isn't found, check api and add to 
        if result.Error == gorm.ErrRecordNotFound {
		upcitemdb.GetBarcode(barcode)
		result = db.DB.Where("barcode = ?", barcode).First(&item)               
        }

	// If there is a connection error
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
                return returnError(c, 400, messages.ErrorWithConnection)
        }
	
	// Search Ownership by barcode
	var ownerships []models.Ownership
	result = db.DB.Where("item_barcode = ? AND item_owner = ?", barcode, uid).Find(&ownerships)

	// If no ownership exists, create ownership
	if len(ownerships) == 0 {
		_, err = createOwnership(uid, barcode)
		if err != nil {return returnError(c, 400, err.Error())}
		return c.Status(200).JSON(
			fiber.Map{
				"success":true,
				"message":"Created new ownership",
				"title":item.Name,
				"barcode":item.Barcode,
				"brand":item.Brand,
				"image":item.Image,
				"owner":uid})
	}

	// If ownerships exist, return as slice
	var ownershipResponses []structs.OwnershipResponse
	for _, ownership := range ownerships {
		ownershipResponse := getOwnershipReponse(ownership)
		ownershipResponses = append(ownershipResponses, ownershipResponse)	
	}

	return c.Status(200).JSON(
                        fiber.Map{
                                "success":true,
                                "message":"Item found",       
				"item":item.Name,
				"brand":item.Brand,
				"image":item.Image,
				"owner":uid,
				"ownership":ownershipResponses})
}

/*
IncrementOwnership increases the ownerships quantity by the designated value

@param c *fiber.Ctx
*/
func ChangeQuantity(c *fiber.Ctx) error {
        // Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}
  
	// Initialize variables
        userUID := data["uid"]
	ownershipUID := c.Query("ownershipUID")
	amountStr := c.Query("amount")
	changeType := c.Params("type")

	// Convert amount to int
	amount, err := strconv.Atoi(amountStr)
	if err != nil {return returnError(c, 400, messages.ConversionError)}
	if amount < 0 {return returnError(c, 400, messages.NegativeError)}

	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Valide and retreive the ownership
	var ownership models.Ownership
	result := db.DB.Where("ownership_uid = ? AND item_owner = ?", ownershipUID, userUID).First(&ownership)
	code, err = recordExists("Ownership", result)
	if err != nil {return returnError(c, code, err.Error())}

	// Check type of change
	switch changeType {
	case "increment":
		ownership.ItemQuantity += amount
	case "decrement":
		ownership.ItemQuantity -= amount
		if ownership.ItemQuantity < 0 {ownership.ItemQuantity = 0}
	case "set":
		ownership.ItemQuantity = amount;
	default:
		return returnError(c, 400, "Invalid change type") // TODO message
	}

	// Save new amount to the database and create response
	db.DB.Save(&ownership)
	ownershipResponse := getOwnershipReponse(ownership)

	// Return success
	return c.Status(200).JSON(
                        fiber.Map{
                                "success":true,
                                "message":"Item found",       
                               	"ownership": ownershipResponse})
}

func DeleteOwnership(c *fiber.Ctx) error {
        // Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}
  
	// Initialize variables
        userUID := data["uid"]
	ownershipUID := c.Query("ownershipUID")
	
	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Validate ownership
	var ownership models.Ownership
	result := db.DB.Where("ownership_uid = ? AND item_owner = ?", ownershipUID, userUID).First(&ownership)
	code, err = recordExists("Ownership", result)
	if err != nil {return returnError(c, code, err.Error())}
	
	db.DB.Delete(&ownership)

	// Check for errors after the delete operation
	if result := db.DB.Delete(&ownership); result.Error != nil {
    		return returnError(c, 500, messages.ErrorDeletingOwnership)
	}

	// Ownership successfully deleted
	return returnSuccess(c, "Ownership deleted successfully")
}

func EditOwnership(c *fiber.Ctx) error {
        // Parse request into data map
	fmt.Println("Start edit ownership")
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}
	fmt.Println("Request parsed")

	// Initialize variables
        userUID := data["uid"]
	ownershipUID := c.Query("ownershipUID")
	fmt.Println("Variables initialized")

	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}
	fmt.Println("Token validated")

	// Validate ownership
	var ownership models.Ownership
	result := db.DB.Where("ownership_uid = ? AND item_owner = ?", ownershipUID, userUID).First(&ownership)
	code, err = recordExists("Ownership", result)
	if err != nil {return returnError(c, code, err.Error())}
	fmt.Println("Ownership validated")

	// Add new fields
	ownership.CustomItemName = c.Query("custom_item_name")
	ownership.CustItemImg = c.Query("custom_item_img")
	ownership.OwnedCustDesc = c.Query("custom_item_description")
	ownership.ItemTags = c.Query("item_tags")
	fmt.Println("Fields added")

	db.DB.Save(&ownership)
	fmt.Println("DB saved")

	// Ownership successfully updated
	return returnSuccess(c, "Ownership updated") // TODO message
}

func CreateOwnership(c *fiber.Ctx) error {
        // Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}
  
	// Initialize variables
        userUID := data["uid"]
	barcode := c.Query("barcode")
	
	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}
	
	ownership, err := createOwnership(userUID, barcode)
	if err!= nil{return returnError(c, code, err.Error())}

	return c.Status(200).JSON(
                        fiber.Map{
                                "success":true,
                                "message":"Ownership created", // TODO messages       
                               	"ownershipUID": ownership.OwnershipUID})
}

func SetOwnershipLocation(c *fiber.Ctx) error{
        // Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}
  
	// Initialize variables
        userUID := data["uid"]	
	locationQR := c.Query("location_qr")
	ownershipUID := c.Query("ownershipUID")

	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Validate the QR code
	var location models.Location
	result := db.DB.Where("location_qr = ? AND location_owner = ?", locationQR, userUID).First(&location)
	code, err = recordExists("Location QR", result)
	if err != nil {return returnError(c, code, err.Error())}

	// Validate the ownership
	var ownership models.Ownership
	result = db.DB.Where("ownership_uid = ? AND item_owner = ?", ownershipUID, userUID).First(&ownership)
	code, err = recordExists("Ownership", result)
	if err != nil {return returnError(c, code, err.Error())}

	// Set the location and save
	ownership.ItemLocation = location.LocationUID
	db.DB.Save(&ownership)

	// return success
	return returnSuccess(c, "Ownership set in " + location.LocationName) // TODO make message
}


