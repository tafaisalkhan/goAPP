package userdetail

import (
	"database/sql"
	"fmt"
	"strings"
)

var userDetailColumns = []string{
	"user_id",
	"user_name",
	"first_name",
	"last_name",
	"email",
	"enabled",
	"role",
	"password",
	"parent_id",
	"secret",
	"mfa_enabled",
	"type",
	"is_staff_member",
	"organization",
	"dob",
	"gender",
	"phone",
	"openstack_user_id",
	"status",
	"active",
	"image",
	"updated_by",
	"created_at",
	"updated_at",
	"hystax_id",
	"role_name",
	"openstack_default_project_id",
	"openstack_id",
	"otp_enabled",
	"description",
	"stripe_id",
	"card_detail",
	"address",
	"allow_credit",
	"city",
	"code_expiration",
	"country",
	"agreed_terms_version",
	"email_verified",
	"is_enforced",
	"state",
	"verification_code",
	"verify_phone",
	"allow_free_credit",
	"credit_limit_date",
	"card_exemption",
	"current_step",
	"card_excemption",
	"creadit_limit_date",
	"current_agreed_terms_version",
	"enforce_enabled",
	"firstname",
	"lastname",
	"test",
	"username",
	"is_password_reset",
	"emirates_id",
	"fax",
	"phone_number",
	"pobox",
	"tenant_id",
	"ntn_no",
}

func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_detail (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			` + createColumnDDL() + `
		)
	`)
	if err != nil {
		return err
	}

	for _, column := range userDetailColumns {
		stmt := fmt.Sprintf("ALTER TABLE user_detail ADD COLUMN IF NOT EXISTS %s TEXT NULL", quoteColumn(column))
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func createColumnDDL() string {
	parts := make([]string, 0, len(userDetailColumns))
	for _, column := range userDetailColumns {
		parts = append(parts, fmt.Sprintf("%s TEXT NULL", quoteColumn(column)))
	}
	return strings.Join(parts, ",\n			")
}

func quoteColumn(name string) string {
	return "`" + name + "`"
}
