package controller

import (
	"WIG-Server/db"
	"WIG-Server/messages"
	"WIG-Server/models"
	"strconv"
	"github.com/gofiber/fiber/v2"
)


func CreateLocation(c *fiber.Ctx) error {	
        // Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}
  
	// Initialize variables
        userUID := data["uid"]	
	locationQR := c.Query("location_qr")
	locationName := c.Query("location_name")
	locationType := c.Params("type")
	
	// Check location type exists
	if locationType != "bin" && locationType != "bag" && locationType != "location" {
		return returnError(c, 400, "Location type is invalid") // TODO make message 
	}

	// convert uid to uint
	userUIDInt, err := strconv.ParseUint(userUID, 10, 64)
	if err != nil {return returnError(c, 400, messages.ConversionError)}

	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Check for empty fields 
	if locationQR == "" {return returnError(c, 400, "QR Location required")} // TODO make message 
	if locationName == "" {return returnError(c, 400, "Location name required")} // TODO make message
	
	// Validate location QR code is not in use
	var location models.Location
	result := db.DB.Where("location_qr = ? AND location_owner = ?", locationQR, userUID).First(&location)
	code, err = recordNotInUse("Location QR", result)
	if err != nil {return returnError(c, code, err.Error())}

	// Valide location name is not in use
	result = db.DB.Where("location_name = ? AND location_owner = ?", locationName, userUID).First(&location)
	code, err = recordNotInUse("Location Name", result)
	if err != nil {return returnError(c, code, err.Error())}

	// create location
	location = models.Location{
		LocationName: locationName,
		LocationOwner: uint(userUIDInt),
		LocationType: locationType,
		LocationQR: locationQR,
	}

	db.DB.Create(&location)

	return returnSuccess(c, "location added successfully") // TODO make message
}

func SetLocationLocation(c *fiber.Ctx) error{
        // Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}
  
	// Initialize variables
        userUID := data["uid"]	
	locationUID := c.Query("location_uid")
	setLocationUID := c.Query("set_location_uid")

	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Verify locations are not the same
	if locationUID == setLocationUID{return returnError(c, 400, "Cannot set location as self")} // TODO message

	// Validate the QR code
	var location models.Location
	result := db.DB.Where("location_uid = ? AND location_owner = ?", locationUID, userUID).First(&location)
	code, err = recordExists("Location QR", result)
	if err != nil {return returnError(c, code, err.Error())}

	// Validate the ownership
	var setLocation models.Location
	result = db.DB.Where("location_uid = ? AND location_owner = ?", setLocationUID, userUID).First(&setLocation)
	code, err = recordExists("Location", result)
	if err != nil {return returnError(c, code, err.Error())}

	// Set the location and save
	location.LocationLocation = &setLocation.LocationUID
	db.DB.Save(&location)

	// return success
	return returnSuccess(c, location.LocationName + " set in " + setLocation.LocationName) // TODO make message
}

func EditLocation(c *fiber.Ctx) error {
        // Parse request into data map
        var data map[string]string
        err := c.BodyParser(&data)
	if err != nil {return returnError(c, 400, messages.ErrorParsingRequest)}

	// Initialize variables
        userUID := data["uid"]
	locationUID := c.Query("locationUID")

	// Validate Token
	code, err := validateToken(c, data["uid"], data["token"])	
	if err != nil {return returnError(c, code, err.Error())}

	// Validate ownership
	var location models.Location
	result := db.DB.Where("location_uid = ? AND location_owner = ?", locationUID, userUID).First(&location)
	code, err = recordExists("Ownership", result)
	if err != nil {return returnError(c, code, err.Error())}

	// Add new fields
	location.LocationName = c.Query("location_name")
	location.LocationDescription = c.Query("location_description")
	location.LocationTags = c.Query("location_tags")

	db.DB.Save(&location)

	// Ownership successfully updated
	return returnSuccess(c, "Ownership updated") // TODO message
}

