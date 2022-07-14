package models

type Settings struct {
	Model

	PrivateJWK EncryptedAtRestBytes
	PublicJWK  []byte

	LowercaseMin int `gorm:"default:0"`
	UppercaseMin int `gorm:"default:0"`
	NumberMin    int `gorm:"default:0"`
	SymbolMin    int `gorm:"default:0"`
	LengthMin    int `gorm:"default:8"`
}
