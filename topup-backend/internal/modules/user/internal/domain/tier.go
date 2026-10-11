// Package domain berisi entity dan aturan bisnis murni modul user.
package domain

// Kode tier (PRC-03). Tier RESELLER_* hanya untuk pemegang role reseller.
const (
	TierGuest            = "GUEST"
	TierMember           = "MEMBER"
	TierResellerSilver   = "RESELLER_SILVER"
	TierResellerGold     = "RESELLER_GOLD"
	TierResellerPlatinum = "RESELLER_PLATINUM"
)

// Tier adalah tier harga bawaan yang di-seed.
type Tier struct {
	Code      string
	Name      string
	Rank      int16 // makin tinggi makin murah
	IsDefault bool  // tier member baru
}

// SystemTiers adalah tier bawaan. Seed hanya menambah yang belum ada, sehingga
// perubahan admin (nama, auto upgrade) tidak tertimpa.
var SystemTiers = []Tier{
	{Code: TierGuest, Name: "Guest", Rank: 0},
	{Code: TierMember, Name: "Member", Rank: 1, IsDefault: true},
	{Code: TierResellerSilver, Name: "Reseller Silver", Rank: 2},
	{Code: TierResellerGold, Name: "Reseller Gold", Rank: 3},
	{Code: TierResellerPlatinum, Name: "Reseller Platinum", Rank: 4},
}
