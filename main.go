package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	gomail "gopkg.in/gomail.v2"
)

type Prospect struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
	Company   string
	Projects  []string
}

type EmailTemplateData struct {
	RecipientName        string
	CompanyName          string
	Date                 string
	Time                 string
	Location             string
	ContactName          string
	ContactNumber        string
	ConfirmationDeadline string
}

func main() {
	// Load variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	startDate := time.Date(2023, time.December, 4, 10, 0, 0, 0, time.UTC)

	// Interval between appointments
	interval := 2 * time.Hour

	prospects := loadProspectsFromCSV("data2.csv")

	// Number of prospects to schedule
	numProspects := len(prospects)

	// Generate the schedule
	schedule, deadlines := generateSchedule(startDate, interval, numProspects)

	for index, prospect := range prospects {
		appointmentTime := schedule[index]
		deadline := deadlines[index]

		sendEmail(
			prospect.Email,
			parseEmailTemplate(
				EmailTemplateData{
					RecipientName:        prospect.FirstName,
					CompanyName:          os.Getenv("COMPANY_NAME"),
					Date:                 appointmentTime.Format("Monday, January 02, 2006"),
					Time:                 appointmentTime.Format("03:04 PM"),
					Location:             os.Getenv("COMPANY_ADDRESS"),
					ContactName:          os.Getenv("CONTACT_NAME"),
					ContactNumber:        os.Getenv("CONTACT_NUMBER"),
					ConfirmationDeadline: deadline.Format("Monday, January 02, 2006") + " before 3:00 PM",
				}),
		)

		writeTemplateFile(
			"output/"+fmt.Sprint(index)+"-"+prospect.FirstName+"-"+prospect.LastName+".html",
			parseEmailTemplate(
				EmailTemplateData{
					RecipientName:        prospect.FirstName,
					CompanyName:          os.Getenv("COMPANY_NAME"),
					Date:                 appointmentTime.Format("Monday, January 02, 2006"),
					Time:                 appointmentTime.Format("03:04 PM"),
					Location:             os.Getenv("COMPANY_ADDRESS"),
					ContactName:          os.Getenv("CONTACT_NAME"),
					ContactNumber:        os.Getenv("CONTACT_NUMBER"),
					ConfirmationDeadline: deadline.Format("Monday, January 02, 2006") + " before 3:00 PM",
				}),
		)
	}
}

func sendEmail(email string, template []byte) {
	fmt.Println("Sending email to: ", email)
	message := gomail.NewMessage()
	message.SetHeader("From", os.Getenv("SMTP_EMAIL"))
	message.SetHeader("To", email)
	message.SetHeader("Subject", os.Getenv("EMAIL_SUBJECT"))

	// Set the HTML body
	message.SetBody("text/html", string(template))

	// Create a new SMTP client
	dialer := gomail.NewDialer(os.Getenv("SMTP_HOST"), 587, os.Getenv("SMTP_EMAIL"), os.Getenv("SMTP_PASSWORD"))

	// Send the email
	if err := dialer.DialAndSend(message); err != nil {
		fmt.Println("Error sending email: ", err)
		return
	}

	fmt.Println("Email sent successfully!")
}

func readCSVfile(filePath string) [][]string {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file: ", err)
		return [][]string{}
	}

	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read the header line to skip it
	_, err = reader.Read()
	if err != nil {
		fmt.Println("Error reading CSV header:", err)
		return [][]string{}
	}

	// Read all records from the CSV file
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV: ", err)
		return [][]string{}
	}

	return records
}

func loadProspectsFromCSV(filePath string) []Prospect {
	var prospects []Prospect

	var records = readCSVfile(filePath)

	// Iterate over the records and convert each record to a Prospect
	for _, record := range records {
		fullName := record[0]

		firstName, lastName := splitFullName(fullName)
		projects := parseCommaSeparatedString(record[3])

		prospect := Prospect{
			FirstName: firstName,
			LastName:  lastName,
			Email:     record[1],
			Phone:     record[2],
			Company:   record[4],
			Projects:  projects,
		}

		prospects = append(prospects, prospect)
	}
	return prospects
}

func splitFullName(fullName string) (firstName, lastName string) {
	names := strings.Fields(fullName)

	if len(names) > 0 {
		firstName = names[0]
		if len(names) > 1 {
			lastName = strings.Join(names[1:], " ")
		}
	}
	return firstName, lastName
}

func parseCommaSeparatedString(inputString string) []string {
	return strings.Split(inputString, ",")
}

func readHTMLTemplate() string {
	// Open the file
	htmlContent, err := os.ReadFile("mail-template.html")
	if err != nil {
		fmt.Println("Error opening file: ", err)
		return ""
	}

	emailTemplate := string(htmlContent)
	return emailTemplate
}

func parseEmailTemplate(data EmailTemplateData) []byte {
	tmpl, err := template.New("emailTemplate").Parse(readHTMLTemplate())
	if err != nil {
		fmt.Println("Error parsing email template: ", err)
	}

	// Create a buffer to store the rendered template
	var templateBuffer bytes.Buffer

	// Execute the template by passing the prospect data into the template
	err = tmpl.Execute(&templateBuffer, data)
	if err != nil {
		fmt.Println("Error executing template: ", err)
	}

	templateBytes := templateBuffer.Bytes()
	return templateBytes
}

func writeTemplateFile(outputFile string, data []byte) {

	// Write the template to the file
	err := os.WriteFile(outputFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing to file: ", err)
		return
	}

	fmt.Println("Email template written to file: ", outputFile)
}

func generateSchedule(start time.Time, interval time.Duration, numProspects int) (schedule, deadlines []time.Time) {

	currentTime := start

	for i := 0; i < numProspects; i++ {
		// Check if the current time is after 4:00 PM
		if currentTime.Hour() >= 16 {
			// If yes, schedule the appointment for the next day at 10:00 AM
			currentTime = currentTime.AddDate(0, 0, 1)
			currentTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 10, 0, 0, 0, time.UTC)
		}

		// Add the current time to the schedule
		schedule = append(schedule, currentTime)

		// Calculate the deadline as one day before the schedules appointment
		deadline := currentTime.AddDate(0, 0, -1)
		deadlines = append(deadlines, deadline)

		// Move to the next appointment time
		currentTime = currentTime.Add(interval)

	}
	return schedule, deadlines
}
