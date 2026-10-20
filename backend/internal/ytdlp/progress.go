package ytdlp

import (
	"strconv"
	"strings"
)

// Progress is one parsed line of yt-dlp's progress template.
type Progress struct {
	Status     string  `json:"status"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Speed      float64 `json:"speed"`
	ETA        int     `json:"eta"`
}

// ParseProgress parses a line produced by the --progress-template value.
//
// The template emits
//
//	dl|<status>|<downloaded>|<total>|<total_estimate>|<speed>|<eta>
//
// while yt-dlp substitutes "NA" for unknown numbers. Short lines that omit the
// estimate field are accepted as well and are read as
//
//	dl|<status>|<downloaded>|<total>|<speed>|<eta>
func ParseProgress(line string) (Progress, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Progress{}, false
	}

	parts := strings.Split(line, "|")
	if len(parts) < 6 || parts[0] != "dl" {
		return Progress{}, false
	}

	p := Progress{Status: parts[1]}

	nums := parts[2:]
	switch len(nums) {
	case 4: // no estimate column
		p.Downloaded = parseInt(nums[0])
		p.Total = parseInt(nums[1])
		p.Speed = parseFloat(nums[2])
		p.ETA = int(parseFloat(nums[3]))
	case 5: // full template
		p.Downloaded = parseInt(nums[0])
		p.Total = parseInt(nums[1])
		estimate := parseInt(nums[2])
		p.Speed = parseFloat(nums[3])
		p.ETA = int(parseFloat(nums[4]))
		if p.Total == 0 && estimate > 0 {
			p.Total = estimate
		}
	default:
		return Progress{}, false
	}

	return p, true
}

func parseInt(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "NA" || s == "None" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return 0
		}
		return int64(f)
	}
	return n
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "NA" || s == "None" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
