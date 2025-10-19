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
