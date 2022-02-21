package commands

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/orblazer/harbor-cli/api"
)

var (
	// Original : `(?i)^(?:((?:(?:(?:[0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}(?:[0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])|(?:[a-z][a-z0-9\-]{0,48}[a-z0-9]\.){1,4}(?:[a-z]|[a-z][a-z0-9\-]{0,48}[a-z0-9])|localhost)(?::[0-9]+)?)/)?([0-9a-z_-]{1,40}(?:/[0-9a-z_-]{1,40})*)(?::([a-z0-9][a-z0-9._-]{1,38}[a-z0-9]))?(?:@sha256:([0-9a-f]{64}))?$``
	imageRe    = regexp.MustCompile(`(?i)^(?P<image>[0-9a-z_-]{1,40}(?:/[0-9a-z_-]{1,40})*)(?::(?P<tag>[a-z0-9][a-z0-9._-]{1,38}[a-z0-9]))?(?:@(?P<digest>sha256:[0-9a-f]{64}))?$`)

	checkInterval      = 3 * time.Second
	supportedMimeTypes = "application/vnd.security.vulnerability.report; version=1.1, application/vnd.scanner.adapter.vuln.report.harbor+json; version=1.0"
)

type scanReport struct {
	ID              string `json:"report_id"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	Status          string `json:"scan_status"`
	CompletePercent int    `json:"complete_percent"`
	Duration        int    `json:"duration"`
	Scanner         struct {
		Name    string `json:"name"`
		Vendor  string `json:"vendor"`
		Version string `json:"version"`
	} `json:"scanner"`
	Severity string `json:"severity"`
	Summary  struct {
		Total   int `json:"total"`
		Fixable int `json:"fixable"`
		Summary struct {
			Critical   int `json:"Critical"`
			High       int `json:"High"`
			Medium     int `json:"Medium"`
			Low        int `json:"Low"`
			Negligible int `json:"Negligible"`
			None       int `json:"None"`
		} `json:"summary"`
	} `json:"summary"`
}

type summary struct {
	ProjectId    int                   `json:"project_id"`
	Digest       string                `json:"digest"`
	ScanOverview map[string]scanReport `json:"scan_overview"`
}

type severity int

const (
	None severity = iota
	Low
	Medium
	High
	Critical
)

func Scan(c *api.Client, registryUrl, rSeverity string, args []string) {
	if len(args) < 1 {
		log.Fatal("[ERROR] missing image argument. Usage: harbor-cli scan <image>")
	}
	maxSevName, maxSeverity := parseSeverity(rSeverity)

	// Remove registry from image
	re := regexp.MustCompile(fmt.Sprintf("^(https?://)?%s/", registryUrl))
	image := re.ReplaceAllString(args[0], "")

	// Parse image
	project, repository, reference := parseImage(image)

	// Launch scan
	err := runScan(c, project, repository, reference)
	if err != nil && err.Error() != "CONFLICT: a previous scan process is Pending" {
		log.Fatal("[ERROR] ", err)
	}

	log.Println("Scanning image...")

	// Retrieve summary
	var sevName string
	var severity severity

	ticker := time.NewTicker(checkInterval)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			res, err := getSummary(c, project, repository, reference)
			if err != nil {
				log.Fatal("[ERROR] ", err)
			}

			// Retrieve report
			var report scanReport
			var ok bool
			if report, ok = res.ScanOverview["application/vnd.security.vulnerability.report; version=1.1"]; ok {
			} else if report, ok = res.ScanOverview["application/vnd.scanner.adapter.vuln.report.harbor+json; version=1.0"]; ok {
			} else {
				return
			}

			if report.Status == "Success" {
				sevName, severity = parseSeverity(report.Severity)

				// Print result
				log.Println("+===============================================+")
				log.Println("|                  Scan report                  |")
				log.Println("+===============================================+")
				// Print artifact url
				log.Printf("| Artifact url: %sharbor/projects/%d/repositories/%s/artifacts/%s\n\n", c.ClientUrl, res.ProjectId,
					repository, res.Digest)
				log.Println("|")
				log.Printf("| Vulnerability Severity: %s", sevName)
				printSummary(report)

				close(quit)
			} else if report.Status == "Error" {
				close(quit)
			}
		case <-quit:
			ticker.Stop()

			if severity >= maxSeverity {
				s := fmt.Sprintf("Severity: %s, Max severity: %s", sevName, maxSevName)

				log.Println("+===============================================+")
				log.Println("| /!\\ The max severity level is reached !       |")
				log.Printf("|  %s%s  |", s, strings.Repeat(" ", 43-len(s)))
				log.Println("+===============================================+")
				os.Exit(1)
			}
			return
		}
	}
}

func parseSeverity(severity string) (string, severity) {
	switch severity {
	case "None":
		return "None", None
	case "Low":
		return "Low", Low
	case "Medium":
		return "Medium", Medium
	case "High":
		return "High", High
	case "Critical":
		return "Critical", Critical
	default:
		log.Fatalln("[ERROR] invald severity argument. Possible values:  None, Low, Medium, High, Critical")
	}
	return "", -1
}

func parseImage(image string) (project, repository, reference string) {
	// Match image
	matches := imageRe.FindStringSubmatch(image)
	if len(matches) < 2 {
		log.Fatal("[ERROR] invalid image. Usage: harbor-cli scan <image>")
	}

	// Extract project name and repository
	firstSlash := strings.Index(matches[1], "/")
	if firstSlash == -1 {
		log.Fatal("[ERROR] invalid image. The image must contains an project and an repository (eg. 'project/repository')")
	}
	project = matches[1][:firstSlash]
	repository = url.PathEscape(matches[1][firstSlash+1:])

	// Reference from digest
	if matches[3] != "" {
		reference = matches[3]
	} else if matches[2] != "" { // Reference from tag
		reference = matches[2]
	} else { // Fallback
		reference = "latest"
	}

	return
}

func runScan(c *api.Client, project, repository, reference string) error {
	req, err := c.CreateRequest("POST", fmt.Sprintf("/projects/%s/repositories/%s/artifacts/%s/scan", project, repository, reference), nil)
	if err != nil {
		return err
	}

	if err := c.SendRequest(req, nil); err != nil {
		if err.Error() != "FORBIDDEN: forbidden" { // Convert to more readable
			return fmt.Errorf("FORBIDDEN: Missing 'create' action on 'scan' resource (/project/%s/scan)", project)
		} else if err.Error() != "EOF" {
			return err
		}
	}

	return nil
}

func getSummary(c *api.Client, project, repository, reference string) (summary, error) {
	req, err := c.CreateRequest("GET", fmt.Sprintf("/projects/%s/repositories/%s/artifacts/%s?with_scan_overview=true", project, repository, reference), nil)
	if err != nil {
		return summary{}, err
	}

	req.Header.Add("X-Accept-Vulnerabilities", supportedMimeTypes)

	var res summary
	if err := c.SendRequest(req, &res); err != nil {
		if err.Error() != "FORBIDDEN: forbidden" { // Convert to more readable
			err = fmt.Errorf("FORBIDDEN: Missing 'read' action on 'artifact' resource (/project/%s/artifact)", project)
		}

		return summary{}, err
	}

	return res, nil
}

func printSummary(report scanReport) {
	log.Printf("| Total: %d (UNKNOWN: %d, LOW: %d, MEDIUM: %d, HIGH: %d, CRITICAL: %d)", report.Summary.Total,
		report.Summary.Summary.None, report.Summary.Summary.Low, report.Summary.Summary.Medium, report.Summary.Summary.High,
		report.Summary.Summary.Critical)
	log.Printf("| *Fixable: %d", report.Summary.Fixable)

	log.Println("|")
	log.Printf("| Scanned by: %s@%s", report.Scanner.Name, report.Scanner.Version)
	log.Printf("| Duration: %s", time.Duration(report.Duration)*time.Second)
}
