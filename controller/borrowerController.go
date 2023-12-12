package controller

import (
	"WIG-Server/db"
	"WIG-Server/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

/**
* Creates a borrower and adds it to the database.
*
* @param c The Fiber context containing the HTTP request and response objects.
*
* @return error The error message, if there is any.
 */
func CreateBorrower(c *fiber.Ctx) error {
	// Initialize variables
	user := c.Locals("user").(models.User)
	borrowerName := c.Query("borrower")

	// Check for empty fields
	if borrowerName == "" {return Error(c, 400, "The borrower field is empty")}

	// Validate location QR code is not in use
	var borrower models.Borrower
	result := db.DB.Where("borrower_name = ? AND borrower_owner = ?", borrowerName, user.UserUID).First(&borrower)
	code, err := recordNotInUse("Borrower Name", result)
	if err != nil {return Error(c, code, err.Error())}

	// create location
	borrower = models.Borrower{
		BorrowerName:  borrowerName,
		BorrowerOwner: user.UserUID,
	}

	db.DB.Create(&borrower)

	borrowerDTO := DTO("borrower", borrower)

	return Success(c, "Borrower created", borrowerDTO)
}

/*
* Sets a list of ownerships to be checked out to a borrower.
*
* @param c The Fiber context containing the HTTP request and response objects.
*
* @return error The error message, if there is any.
*/
func CheckoutItem(c *fiber.Ctx) error {
	// Initialize variables
	user := c.Locals("user").(models.User)
	var request BorrowerRequest
	borrowerUID, err := strconv.ParseUint(c.Query("borrowerUID"), 10, 64)

	if err != nil {
		return Error(c, 400, "There was an error converting borrowerUID")
	}

	err = c.BodyParser(&request)
	if err != nil {return Error(c, 400, "There was an error parsing JSON")}

	// Get borrower	
	var borrower models.Borrower
	result := db.DB.Where("borrower_uid = ? AND borrower_owner = ?", borrowerUID, user.UserUID).First(&borrower)

	code, err := RecordExists("Borrower UID", result)
	if err != nil {return Error(c, code, err.Error())}

	success := 0
	var successfulOwnerships []int

	for _, ownership := range request.Ownerships {		
		var item models.Ownership
		result = db.DB.Where("ownership_uid = ? AND item_owner = ?", ownership, user.UserUID).First(&item)
		
		_, err := RecordExists("Ownership", result)
		if err == nil {
			item.ItemBorrower = uint(borrowerUID)
			db.DB.Save(&item)
			preloadOwnership(&item)
			successfulOwnerships = append(successfulOwnerships, ownership)
			success++
		}
	}

	if success == 0 {
		return Error(c, 400, "Failed to checkout ownerships")
	}

	ownershipsDTO := DTO("ownerships", successfulOwnerships)	
	return Success(c, "Checked out", ownershipsDTO)
}

/*
* Sets returns checked out items to original owners within the list.
*
* @param c The Fiber context containing the HTTP request and response objects.
*
* @return error The error message, if there is any.
*/

func CheckinItem(c *fiber.Ctx) error {
	// Initialize variables
	user := c.Locals("user").(models.User)
	var request BorrowerRequest

	err := c.BodyParser(&request)
	if err != nil {return Error(c, 400, "There was an error parsing JSON")}

	success := 0
	var successfulOwnerships []int

	for _, ownership := range request.Ownerships {		
		var item models.Ownership
		result := db.DB.Where("ownership_uid = ? AND item_owner = ?", ownership, user.UserUID).First(&item)
		
		_, err := RecordExists("Ownership", result)
		if err == nil {
			item.ItemBorrower = uint(1)
			db.DB.Save(&item)
			preloadOwnership(&item)
			successfulOwnerships = append(successfulOwnerships, ownership)
			success++
		}
	}

	if success == 0 {
		return Error(c, 400, "Failed to checkout ownerships")
	}

	ownershipsDTO := DTO("ownerships", successfulOwnerships)	
	return Success(c, "Checked in", ownershipsDTO)

}

/*
* Returns all borrowers associated with user.
*
* @param c The Fiber context containing the HTTP request and response objects.
*
* @return error The error message, if there is any.
*/
func GetBorrowers(c *fiber.Ctx) error{
	// Initialize variables
	user := c.Locals("user").(models.User)

	// Get borrower	
	var borrowers []models.Borrower
	db.DB.Where("borrower_owner = ?", user.UserUID).Find(&borrowers)

	if len(borrowers) == 0 {
		return Error(c, 404, "No borrowers found")
	}

	borrowersDTO := DTO("borrowers", &borrowers)

	return Success(c, "Borrowers returned", borrowersDTO)
}


type BorrowerRequest struct {
	Ownerships []int `json:"ownerships"` 
}
