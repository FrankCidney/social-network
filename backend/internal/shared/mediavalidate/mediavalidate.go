package mediavalidate

const MaxImageSize   = 5 << 20 // 5 MB

var AllowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
}

// IsAllowedImageBytes checks magic bytes for JPEG, PNG, and GIF. We do this independently of the filename extension as a second layer of
// validation. A renamed .exe for example, uploaded as .jpg, should be rejected.
func IsAllowedImageBytes(b []byte) bool {
	// If the file is fewer than 4 bytes, it can't be a valid JPEG, PNG or GIF
	if len(b) < 4 {
		return false
	}

	// Every JPEG begins with the hex sequence FF D8 FF
	if b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF {
		return true
	}

	// Every PNG begins with the hex sequence 89 50 4E 47 0D 0A 1A 0A
	if len(b) >= 8 &&
		b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47 &&
		b[4] == 0x0D && b[5] == 0x0A && b[6] == 0x1A && b[7] == 0x0A {
		return true
	}

	// Every GIF starts with the text GIF87a or GIF89a
	if len(b) >= 6 &&
		b[0] == 'G' && b[1] == 'I' && b[2] == 'F' && b[3] == '8' &&
		(b[4] == '7' || b[4] == '9') && b[5] == 'a' {
		return true
	}

	return false
}
