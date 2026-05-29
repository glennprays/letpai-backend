// Bootstrap-admin is a one-off CLI for creating or repairing the seed
// super admin row. It reuses the same use case the HTTP wizard
// /admin/setup uses so the gate (a usable super admin already exists →
// refuse) lives in one place.
//
// Usage:
//
//	go run ./cmd/bootstrap-admin \
//	    --whatsapp-number=6281234567890 \
//	    --full-name='Glenn' \
//	    --password='something-strong'
//
// Any flag left empty triggers an interactive prompt on stdin. The
// password prompt does NOT mask input (we don't pull in golang.org/x/term
// just for this). Pass --password=… non-interactively or pipe it via
// stdin if you'd rather not have it echoed.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/internal/infrastructure"
	"github.com/glennprays/letpai-backend/internal/repository"
	"github.com/glennprays/letpai-backend/internal/service"
	adminuc "github.com/glennprays/letpai-backend/internal/usecase/admin"
)

func main() {
	whatsapp := flag.String("whatsapp-number", "", "WhatsApp number in E.164 form (no +), e.g. 6281234567890")
	fullName := flag.String("full-name", "", "Display name for the super admin")
	password := flag.String("password", "", "Password (≥8 chars). Will be bcrypt-hashed.")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)
	*whatsapp = promptIfEmpty(reader, "WhatsApp number (E.164, no +): ", *whatsapp)
	*fullName = promptIfEmpty(reader, "Full name: ", *fullName)
	*password = promptIfEmpty(reader, "Password (≥8 chars, will be echoed): ", *password)

	if *whatsapp == "" || *fullName == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "all three values are required")
		os.Exit(2)
	}
	if len(*password) < 8 {
		fmt.Fprintln(os.Stderr, "password must be at least 8 characters")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	db, err := infrastructure.NewPostgresConnection(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to postgres: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	adminRepo := repository.NewPostgresAdminRepository(db)
	passwordSvc := service.NewPasswordService(12)

	// jwtSvc is nil; the CLI passes issueToken=false to BootstrapUseCase
	// so it never tries to mint a token (and never needs the JWT secret
	// to be configured for this binary).
	uc := adminuc.NewBootstrapUseCase(adminRepo, passwordSvc, nil)

	ctx := context.Background()
	result, err := uc.Execute(ctx, &adminuc.BootstrapRequest{
		WhatsAppNumber: *whatsapp,
		FullName:       *fullName,
		Password:       *password,
	}, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created/updated super admin %s for +%s\n", result.AdminID, *whatsapp)
}

// promptIfEmpty returns the existing value when non-empty, otherwise
// prints the prompt and reads a line from r (stripped of trailing
// newlines and surrounding whitespace). Returns "" if reading fails.
func promptIfEmpty(r *bufio.Reader, prompt, existing string) string {
	if existing != "" {
		return existing
	}
	fmt.Fprint(os.Stderr, prompt)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSpace(line)
}
