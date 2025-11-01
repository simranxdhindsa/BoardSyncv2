package utils

import (
	"regexp"
	"strings"
)

// ConvertAsanaHTMLToYouTrackMarkdown converts Asana HTML formatting to YouTrack markdown
func ConvertAsanaHTMLToYouTrackMarkdown(htmlText string) string {
	if htmlText == "" {
		return ""
	}

	text := htmlText

	// Convert HTML paragraphs to line breaks
	text = regexp.MustCompile(`<\/p>\s*<p>`).ReplaceAllString(text, "\n\n")
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", "\n")

	// Convert line breaks
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")

	// Convert bold formatting
	// <strong>text</strong> or <b>text</b> -> *text*
	text = regexp.MustCompile(`<strong>(.*?)</strong>`).ReplaceAllString(text, "*$1*")
	text = regexp.MustCompile(`<b>(.*?)</b>`).ReplaceAllString(text, "*$1*")

	// Convert italic formatting
	// <em>text</em> or <i>text</i> -> _text_
	text = regexp.MustCompile(`<em>(.*?)</em>`).ReplaceAllString(text, "_$1_")
	text = regexp.MustCompile(`<i>(.*?)</i>`).ReplaceAllString(text, "_$1_")

	// Convert strikethrough formatting
	// <s>text</s> or <strike>text</strike> or <del>text</del> -> ~~text~~
	text = regexp.MustCompile(`<s>(.*?)</s>`).ReplaceAllString(text, "~~$1~~")
	text = regexp.MustCompile(`<strike>(.*?)</strike>`).ReplaceAllString(text, "~~$1~~")
	text = regexp.MustCompile(`<del>(.*?)</del>`).ReplaceAllString(text, "~~$1~~")

	// Convert inline code
	// <code>text</code> -> `text`
	text = regexp.MustCompile(`<code>(.*?)</code>`).ReplaceAllString(text, "`$1`")

	// Convert code blocks
	// <pre>code</pre> -> ```\ncode\n```
	text = regexp.MustCompile(`<pre>(.*?)</pre>`).ReplaceAllString(text, "```\n$1\n```")

	// Convert unordered lists
	// <ul><li>item</li></ul> -> * item
	text = regexp.MustCompile(`<ul>\s*`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\s*</ul>`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`<li>(.*?)</li>`).ReplaceAllString(text, "* $1\n")

	// Convert ordered lists
	// <ol><li>item</li></ol> -> 1. item
	olPattern := regexp.MustCompile(`<ol>(.*?)</ol>`)
	text = olPattern.ReplaceAllStringFunc(text, func(match string) string {
		items := regexp.MustCompile(`<li>(.*?)</li>`).FindAllStringSubmatch(match, -1)
		result := ""
		for i, item := range items {
			if len(item) > 1 {
				result += strings.Repeat(" ", 0) + string(rune(i+1)) + ". " + item[1] + "\n"
			}
		}
		return result
	})

	// Convert blockquotes
	// <blockquote>text</blockquote> -> > text
	text = regexp.MustCompile(`<blockquote>(.*?)</blockquote>`).ReplaceAllStringFunc(text, func(match string) string {
		content := regexp.MustCompile(`<blockquote>(.*?)</blockquote>`).FindStringSubmatch(match)
		if len(content) > 1 {
			lines := strings.Split(strings.TrimSpace(content[1]), "\n")
			for i, line := range lines {
				lines[i] = "> " + strings.TrimSpace(line)
			}
			return strings.Join(lines, "\n")
		}
		return match
	})

	// Convert headings
	// <h1>text</h1> -> # text
	// <h2>text</h2> -> ## text
	// <h3>text</h3> -> ### text
	text = regexp.MustCompile(`<h1>(.*?)</h1>`).ReplaceAllString(text, "# $1\n")
	text = regexp.MustCompile(`<h2>(.*?)</h2>`).ReplaceAllString(text, "## $1\n")
	text = regexp.MustCompile(`<h3>(.*?)</h3>`).ReplaceAllString(text, "### $1\n")
	text = regexp.MustCompile(`<h4>(.*?)</h4>`).ReplaceAllString(text, "#### $1\n")
	text = regexp.MustCompile(`<h5>(.*?)</h5>`).ReplaceAllString(text, "##### $1\n")
	text = regexp.MustCompile(`<h6>(.*?)</h6>`).ReplaceAllString(text, "###### $1\n")

	// Convert horizontal rules
	text = strings.ReplaceAll(text, "<hr>", "---\n")
	text = strings.ReplaceAll(text, "<hr/>", "---\n")
	text = strings.ReplaceAll(text, "<hr />", "---\n")

	// Convert links
	// <a href="url">text</a> -> [text](url)
	text = regexp.MustCompile(`<a href="(.*?)">(.*?)</a>`).ReplaceAllString(text, "[$2]($1)")

	// Clean up any remaining HTML tags
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")

	// Clean up multiple newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// Decode HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return strings.TrimSpace(text)
}

// ConvertYouTrackMarkdownToAsanaHTML converts YouTrack markdown to Asana HTML formatting
func ConvertYouTrackMarkdownToAsanaHTML(markdown string) string {
	if markdown == "" {
		return ""
	}

	text := markdown

	// Step 1: Process line by line to handle structure (lists, blockquotes, paragraphs)
	lines := strings.Split(text, "\n")
	var result []string
	inList := false
	listType := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for heading markers first
		if regexp.MustCompile(`^#{1,6}\s+`).MatchString(trimmed) {
			// Close any open list
			if inList {
				if listType == "ul" {
					result = append(result, "</ul>")
				} else {
					result = append(result, "</ol>")
				}
				inList = false
				listType = ""
			}
			result = append(result, line) // Keep as-is for now
			continue
		}

		// Check for horizontal rule
		if regexp.MustCompile(`^(---+|\*\*\*+)$`).MatchString(trimmed) {
			if inList {
				if listType == "ul" {
					result = append(result, "</ul>")
				} else {
					result = append(result, "</ol>")
				}
				inList = false
				listType = ""
			}
			result = append(result, line)
			continue
		}

		// Check for unordered list item
		if regexp.MustCompile(`^[\*\-]\s+`).MatchString(trimmed) {
			item := regexp.MustCompile(`^[\*\-]\s+`).ReplaceAllString(trimmed, "")
			if !inList || listType != "ul" {
				if inList && listType == "ol" {
					result = append(result, "</ol>")
				}
				result = append(result, "<ul>")
				inList = true
				listType = "ul"
			}
			result = append(result, "<li>"+item+"</li>")
		} else if regexp.MustCompile(`^\d+\.\s+`).MatchString(trimmed) {
			// Ordered list item
			item := regexp.MustCompile(`^\d+\.\s+`).ReplaceAllString(trimmed, "")
			if !inList || listType != "ol" {
				if inList && listType == "ul" {
					result = append(result, "</ul>")
				}
				result = append(result, "<ol>")
				inList = true
				listType = "ol"
			}
			result = append(result, "<li>"+item+"</li>")
		} else {
			// Not a list item - close list if open
			if inList {
				if listType == "ul" {
					result = append(result, "</ul>")
				} else {
					result = append(result, "</ol>")
				}
				inList = false
				listType = ""
			}

			// Check for blockquote
			if strings.HasPrefix(trimmed, ">") {
				quoted := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
				result = append(result, "<blockquote>"+quoted+"</blockquote>")
			} else if trimmed != "" {
				// Regular paragraph
				result = append(result, "<p>"+line+"</p>")
			} else {
				// Empty line
				if i > 0 && i < len(lines)-1 {
					result = append(result, "<br>")
				}
			}
		}
	}

	// Close any open lists
	if inList {
		if listType == "ul" {
			result = append(result, "</ul>")
		} else {
			result = append(result, "</ol>")
		}
	}

	text = strings.Join(result, "\n")

	// Step 2: Convert headings
	text = regexp.MustCompile(`(?m)^######\s+(.*?)$`).ReplaceAllString(text, "<h6>$1</h6>")
	text = regexp.MustCompile(`(?m)^#####\s+(.*?)$`).ReplaceAllString(text, "<h5>$1</h5>")
	text = regexp.MustCompile(`(?m)^####\s+(.*?)$`).ReplaceAllString(text, "<h4>$1</h4>")
	text = regexp.MustCompile(`(?m)^###\s+(.*?)$`).ReplaceAllString(text, "<h3>$1</h3>")
	text = regexp.MustCompile(`(?m)^##\s+(.*?)$`).ReplaceAllString(text, "<h2>$1</h2>")
	text = regexp.MustCompile(`(?m)^#\s+(.*?)$`).ReplaceAllString(text, "<h1>$1</h1>")

	// Step 3: Convert horizontal rules
	text = regexp.MustCompile(`(?m)^---+$`).ReplaceAllString(text, "<hr>")
	text = regexp.MustCompile(`(?m)^\*\*\*+$`).ReplaceAllString(text, "<hr>")

	// Step 4: Convert code blocks (must be before inline code)
	text = regexp.MustCompile("```([\\s\\S]*?)```").ReplaceAllString(text, "<pre>$1</pre>")

	// Step 5: Convert inline code
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "<code>$1</code>")

	// Step 6: Convert bold formatting (must be before single asterisk)
	text = regexp.MustCompile(`\*\*([^\*]+)\*\*`).ReplaceAllString(text, "<strong>$1</strong>")
	text = regexp.MustCompile(`\*([^\*\n]+)\*`).ReplaceAllString(text, "<strong>$1</strong>")

	// Step 7: Convert italic formatting
	text = regexp.MustCompile(`_([^_\n]+)_`).ReplaceAllString(text, "<em>$1</em>")

	// Step 8: Convert strikethrough
	text = regexp.MustCompile(`~~([^~]+)~~`).ReplaceAllString(text, "<s>$1</s>")

	// Step 9: Convert links
	text = regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`).ReplaceAllString(text, `<a href="$2">$1</a>`)

	// Wrap in <body> tag as required by Asana API
	text = strings.TrimSpace(text)
	if text != "" {
		text = "<body>" + text + "</body>"
	}

	return text
}
