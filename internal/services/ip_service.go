package services

import "database/sql"

func IsVPN(db *sql.DB, ip string) bool {

	var isVPN bool

	err := db.QueryRow(`
		SELECT is_vpn FROM ip_profiles WHERE ip=$1
	`, ip).Scan(&isVPN)

	if err != nil {
		return false
	}

	return isVPN
}
