package twilio

import "strings"

// GSM 03.38's default alphabet (ESC is reserved for the extension table).
const gsmBasic = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
const gsmExtension = "\f^{}\\[~]|€"

// segments counts septets for GSM-7, otherwise UTF-16 code units. Multipart
// boundaries never split an escape sequence or a surrogate pair.
func segments(body string) int {
	if body == "" {
		return 0
	}
	gsm := true
	total := 0
	for _, r := range body {
		switch {
		case strings.ContainsRune(gsmBasic, r):
			total++
		case strings.ContainsRune(gsmExtension, r):
			total += 2
		default:
			gsm = false
		}
	}
	single, multi := 160, 153
	if !gsm {
		single, multi, total = 70, 67, 0
		for _, r := range body {
			total++
			if r > 0xffff {
				total++
			}
		}
	}
	if total <= single {
		return 1
	}
	count, used := 1, 0
	for _, r := range body {
		width := 1
		if (gsm && strings.ContainsRune(gsmExtension, r)) || (!gsm && r > 0xffff) {
			width = 2
		}
		if used+width > multi {
			count++
			used = 0
		}
		used += width
	}
	return count
}
