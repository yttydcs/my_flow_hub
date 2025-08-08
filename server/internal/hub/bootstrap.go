package hub

import (
	"myflowhub/pkg/database"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Bootstrap ensures the server has a persistent identity in the database.
func (s *Server) Bootstrap() {
	var device database.Device
	err := database.DB.Where("hardware_id = ?", s.HardwareID).First(&device).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		log.Fatal().Err(err).Msg("Failed to query for server's own device record")
	}

	if err == gorm.ErrRecordNotFound {
		log.Info().Str("hardwareID", s.HardwareID).Msg("Server device record not found, creating a new one...")

		tempSecret := "a-very-secure-secret-for-the-server"
		hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(tempSecret), bcrypt.DefaultCost)

		role := database.RoleRelay
		if s.ParentAddr == "" {
			role = database.RoleHub
		}

		newDevice := database.Device{
			HardwareID:    s.HardwareID,
			SecretKeyHash: string(hashedSecret),
			Role:          role,
			Name:          s.HardwareID,
		}

		if err := database.DB.Create(&newDevice).Error; err != nil {
			log.Fatal().Err(err).Msg("Failed to create server's own device record")
		}

		device = newDevice
		s.SecretKey = tempSecret
	} else {
		log.Info().Msg("Server device record found.")
		s.SecretKey = "a-very-secure-secret-for-the-server"
	}

	s.DeviceID = device.DeviceUID
	log.Info().Uint64("deviceID", s.DeviceID).Msg("Server identity bootstrapped successfully")
}
