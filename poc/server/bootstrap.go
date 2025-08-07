package main

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// bootstrap ensures the server has a persistent identity in the database.
func (s *Server) bootstrap() {
	var device Device
	err := DB.Where("hardware_id = ?", s.hardwareID).First(&device).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		log.Fatal().Err(err).Msg("Failed to query for server's own device record")
	}

	// If the server's record doesn't exist, create it.
	if err == gorm.ErrRecordNotFound {
		log.Info().Str("hardwareID", s.hardwareID).Msg("Server device record not found, creating a new one...")

		// In a real app, this secret should be generated securely and stored safely.
		tempSecret := "a-very-secure-secret-for-the-server"
		hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(tempSecret), bcrypt.DefaultCost)

		role := RoleRelay
		if s.parentAddr == "" {
			role = RoleHub
		}

		newDevice := Device{
			HardwareID:    s.hardwareID,
			SecretKeyHash: string(hashedSecret),
			Role:          role,
			Name:          s.hardwareID, // Use hardware ID as name by default
		}

		if err := DB.Create(&newDevice).Error; err != nil {
			log.Fatal().Err(err).Msg("Failed to create server's own device record")
		}

		// Use the newly created device for the running instance
		device = newDevice
		s.secretKey = tempSecret // Store the unhashed secret for this session
	} else {
		// If device exists, we need a way to get the unhashed secret.
		// For this PoC, we'll just use a default one. This is a security shortcut.
		log.Info().Msg("Server device record found.")
		s.secretKey = "a-very-secure-secret-for-the-server"
	}

	// Load the persistent identity into the server instance
	s.deviceID = device.DeviceUID
	log.Info().Uint64("deviceID", s.deviceID).Msg("Server identity bootstrapped successfully")
}
