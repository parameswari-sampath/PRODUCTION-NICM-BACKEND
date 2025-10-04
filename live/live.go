package live

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"mcq-exam/db"
	"mcq-exam/utils"
	"os"
	"time"
)

// ============================================
// PHASE 0 - Initial Setup Functions
// ============================================

// generateToken generates a unique token for a user
func generateToken(userId int) string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// storeTokenInDB stores the token in database
func storeTokenInDB(userId int, token string, mailType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO email_tracking (student_id, email_type, conference_token, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (student_id, email_type)
		DO UPDATE SET conference_token = $3, updated_at = NOW()
	`
	_, err := db.Pool.Exec(ctx, query, userId, mailType, token)
	return err
}

// sendFirstMail sends the first email with token
func sendFirstMail(userId int, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user details
	var name, email string
	query := `SELECT name, email FROM students WHERE id = $1`
	err := db.Pool.QueryRow(ctx, query, userId).Scan(&name, &email)
	if err != nil {
		return fmt.Errorf("failed to get user details: %w", err)
	}

	// Get frontend URL from environment
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	// Create conference link with token
	conferenceLink := fmt.Sprintf("%s/live?token=%s", frontendURL, token)

	// Email body
	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 700px; margin: 0 auto; padding: 20px;">
			<h2 style="color: #2c3e50;">Invitation to the Inaugural Virtual Meeting – International Online Quiz on Cooperatives</h2>

			<p>Dear %s,</p>

			<p><strong>Greetings from Natesan Institute of Cooperative Management (NICM), Chennai!</strong></p>

			<p>In commemoration of the <strong>International Year of Cooperatives</strong> and in alignment with the vision of <strong>"Sahakar Se Samriddhi"</strong> (Prosperity through Cooperation), we are delighted to host the <strong>International Online Quiz on Cooperatives</strong>. This event celebrates the strength of the cooperative movement in fostering inclusive growth, empowerment, and sustainable development across the globe.</p>

			<p>We cordially invite you to join the <strong>Inaugural Virtual Meeting</strong> of the International Online Quiz:</p>

			<div style="background-color: #f8f9fa; padding: 15px; border-left: 4px solid #4CAF50; margin: 20px 0;">
				<p style="margin: 5px 0;"><strong>📅 Date:</strong> 8th October 2025</p>
				<p style="margin: 5px 0;"><strong>🕒 Login Time:</strong> 1:45 PM (IST) onwards</p>
				<p style="margin: 5px 0;"><strong>🎤 Inauguration:</strong> 2:00 PM (IST)</p>
				<p style="margin: 5px 0;"><strong>🔗 Join Link:</strong> <a href="%s" style="color: #4CAF50; font-weight: bold;">Click here to join</a></p>
			</div>

			<h3 style="color: #2c3e50;">Important Instructions for Participants:</h3>
			<ul style="line-height: 1.8;">
				<li>At the end of this inaugural session, you will receive your link for the International Online Quiz.</li>
				<li>The quiz will be conducted between <strong>2:30 PM and 3:30 PM</strong> (your local time).</li>
				<li>Upon completion, you can view your responses, the correct answers, and your overall score.</li>
				<li>All participants will receive a <strong>Participation Certificate</strong>.</li>
				<li>The <strong>Top 10 scorers</strong> will be awarded <strong>Merit Certificates</strong>.</li>
				<li>The <strong>Winner</strong> will be selected based on the highest score and the time taken to complete the quiz (in case of a tie, faster completion time will be considered).</li>
			</ul>

			<p>This international event is not just a competition but also a platform to celebrate the spirit of cooperation and its role in creating a sustainable and equitable world.</p>

			<p>We look forward to your enthusiastic participation and presence in the inaugural session.</p>

			<p style="margin-top: 30px;">With warm regards,</p>
			<p><strong>Dr. U. Homiga</strong><br>
			Event Convenor,<br>
			Natesan Institute of Cooperative Management (NICM), Chennai</p>

			<p style="text-align: center; color: #4CAF50; font-style: italic; margin-top: 30px; font-size: 16px;">
				"Cooperatives: Building a Better World Together"
			</p>
		</div>
	`, name, conferenceLink)

	params := utils.SendEmailParams{
		ToEmail:  email,
		ToName:   name,
		Subject:  "Invitation to the Inaugural Virtual Meeting – International Online Quiz on Cooperatives",
		HTMLBody: htmlBody,
	}

	_, err = utils.SendEmail(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Sent first mail to %s with token", email)
	return nil
}

// ============================================
// PHASE 1 - First Mail Verification
// ============================================

func Phase1FirstMailVerification() {
	log.Println("Phase 1: Starting First Mail Verification process")

	// Get all students from database
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT id FROM students ORDER BY id`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("ERROR: Failed to fetch students: %v", err)
		return
	}
	defer rows.Close()

	var studentIds []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		studentIds = append(studentIds, id)
	}

	if len(studentIds) == 0 {
		log.Println("WARNING: No students found")
		return
	}

	// For each student: generate token, store in DB, send first mail
	sentCount := 0
	for _, userId := range studentIds {
		// Step 1: Generate token
		token := generateToken(userId)

		// Step 2: Store token in DB
		err := storeTokenInDB(userId, token, "firstMail")
		if err != nil {
			log.Printf("ERROR: Failed to store token for user %d: %v", userId, err)
			continue
		}

		// Step 3: Send first mail
		err = sendFirstMail(userId, token)
		if err != nil {
			log.Printf("ERROR: Failed to send first mail to user %d: %v", userId, err)
			continue
		}

		sentCount++
	}

	log.Printf("Phase 1 completed: Sent %d/%d first mails", sentCount, len(studentIds))
}

// getToken extracts token from request
func getToken(request interface{}) string {
	// TODO: Extract token from request
	return ""
}

// verifyTokenWithDB verifies if token exists in database
func verifyTokenWithDB(token string, mailType string) (int, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var userId int
	var attended bool

	query := `
		SELECT student_id, conference_attended
		FROM email_tracking
		WHERE conference_token = $1 AND email_type = $2
	`
	err := db.Pool.QueryRow(ctx, query, token, mailType).Scan(&userId, &attended)
	if err != nil {
		return 0, false, err
	}

	return userId, attended, nil
}

// returnYoutubeUrlFromDB returns YouTube URL from database
func returnYoutubeUrlFromDB(userId int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var videoUrl string
	query := `SELECT video_url FROM event_schedule ORDER BY id DESC LIMIT 1`
	err := db.Pool.QueryRow(ctx, query).Scan(&videoUrl)
	return videoUrl, err
}

// ============================================
// PHASE 2 - Second Mail Sending
// ============================================

func Phase2SecondMailSending() {
	log.Println("Phase 2: Starting Second Mail Sending process")

	// Step 1: Get all users who verified first mail (conference_attended = true)
	userIds, err := getVerifiedUsersFromDB("firstMail")
	if err != nil {
		log.Printf("ERROR: Failed to get verified users: %v", err)
		return
	}

	if len(userIds) == 0 {
		log.Println("WARNING: No verified users found for second mail")
		return
	}

	log.Printf("Found %d verified users for second mail", len(userIds))

	// Step 2: For each verified user: generate token, store in DB, send second mail
	sentCount := 0
	for _, userId := range userIds {
		// Generate token for second mail
		token := generateToken(userId)

		// Store token in DB with mailType = "secondMail"
		err := storeTokenInDB(userId, token, "secondMail")
		if err != nil {
			log.Printf("ERROR: Failed to store second mail token for user %d: %v", userId, err)
			continue
		}

		// Send second mail with token
		err = sendSecondMail(userId, token)
		if err != nil {
			log.Printf("ERROR: Failed to send second mail to user %d: %v", userId, err)
			continue
		}

		sentCount++
	}

	log.Printf("Phase 2 completed: Sent %d/%d second mails", sentCount, len(userIds))
}

// getVerifiedUsersFromDB gets all users who verified first mail
func getVerifiedUsersFromDB(mailType string) ([]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT student_id
		FROM email_tracking
		WHERE email_type = $1 AND conference_attended = true
	`
	rows, err := db.Pool.Query(ctx, query, mailType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIds []int
	for rows.Next() {
		var userId int
		if err := rows.Scan(&userId); err != nil {
			continue
		}
		userIds = append(userIds, userId)
	}

	return userIds, nil
}

// sendSecondMail sends the second email with access code (OTP)
func sendSecondMail(userId int, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user details and access code from DB
	var name, email, accessCode string
	query := `
		SELECT s.name, s.email, et.access_code
		FROM students s
		JOIN email_tracking et ON s.id = et.student_id
		WHERE s.id = $1 AND et.email_type = 'firstMail' AND et.conference_attended = true
	`
	err := db.Pool.QueryRow(ctx, query, userId).Scan(&name, &email, &accessCode)
	if err != nil {
		return fmt.Errorf("failed to get user details: %w", err)
	}

	if accessCode == "" {
		return fmt.Errorf("access code not found for user %d", userId)
	}

	// Get frontend URL from environment
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	// Create URL with otp parameter
	testURL := fmt.Sprintf("%s?otp=%s", frontendURL, accessCode)

	// Email body
	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Test Invitation - SmartMCQ</h2>
			<p>Dear %s,</p>
			<p>Thank you for attending the conference!</p>
			<p>You are now eligible to take the test. Click the link below to start:</p>
			<p><a href="%s" style="background-color: #2196F3; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Start Test</a></p>
			<p>Or use this access code: <strong>%s</strong></p>
			<p>Best regards,<br>SmartMCQ Team</p>
		</div>
	`, name, testURL, accessCode)

	params := utils.SendEmailParams{
		ToEmail:  email,
		ToName:   name,
		Subject:  "Test Invitation - Your Access Code",
		HTMLBody: htmlBody,
	}

	_, err = utils.SendEmail(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Sent second mail to %s with OTP: %s", email, accessCode)
	return nil
}

// ============================================
// PHASE 2 - Second Mail Verification
// ============================================

func Phase2SecondMailVerification() {
	// This is the main phase 2 verification function
}

// validateUserExists checks if user exists in database
func validateUserExists(userId int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM students WHERE id = $1)`
	err := db.Pool.QueryRow(ctx, query, userId).Scan(&exists)
	return exists, err
}

// checkExistingSession checks if user already has a session
func checkExistingSession(userId int) (bool, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sessionId int
	query := `SELECT id FROM sessions WHERE student_id = $1 LIMIT 1`
	err := db.Pool.QueryRow(ctx, query, userId).Scan(&sessionId)

	if err != nil {
		// No session exists
		return false, 0, nil
	}

	return true, sessionId, nil
}

// returnSessionAlreadyCompleted returns error that session already exists
func returnSessionAlreadyCompleted() error {
	return fmt.Errorf("session already completed")
}

// createSession creates a new session for user
func createSession(userId int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sessionId int
	query := `
		INSERT INTO sessions (student_id, created_at)
		VALUES ($1, NOW())
		RETURNING id
	`
	err := db.Pool.QueryRow(ctx, query, userId).Scan(&sessionId)
	return sessionId, err
}

// returnSessionId returns the session ID
func returnSessionId(sessionId int) int {
	return sessionId
}

// ============================================
// PHASE 3 - Start Test
// ============================================

func Phase3StartTest() {
	// This is the main phase 3 function that starts the test
}

// getSessionId extracts session ID from request
func getSessionId(request interface{}) int {
	// TODO: Extract session ID from request
	return 0
}

// startSession marks session as started
func startSession(sessionId int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE sessions SET started = true WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, sessionId)
	return err
}

// markSessionStartTime marks the start time for session
func markSessionStartTime(sessionId int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE sessions SET started_at = NOW() WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, sessionId)
	return err
}
