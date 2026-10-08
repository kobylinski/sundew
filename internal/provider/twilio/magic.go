package twilio

// magicError follows the SMS parameter tables, not the phone-number purchase
// or voice-call tables on the same reference page. Ordinary numbers succeed.
// Reference: https://www.twilio.com/docs/iam/test-credentials#test-sending-an-sms
// Read 2026-10-08. When both fields trigger, sender validation takes precedence.
func magicError(from, to string) (int, string) {
	switch from {
	case "+15005550001":
		return 21212, "The 'From' number is not a valid phone number"
	case "+15005550007":
		return 21606, "The 'From' phone number is not a valid SMS-capable inbound phone number for your account"
	case "+15005550008":
		return 21611, "SMS message queue is full"
	}
	switch to {
	case "+15005550001":
		return 21211, "The 'To' number is not a valid phone number"
	case "+15005550002":
		return 21612, "Message cannot be sent with the current combination of 'To' and/or 'From' parameters"
	case "+15005550003":
		return 21408, "Permission to send an SMS has not been enabled for the region indicated by the 'To' number"
	case "+15005550004":
		return 21610, "Attempt to send to unsubscribed recipient"
	case "+15005550009":
		return 21614, "'To' number is not a valid mobile number"
	}
	return 0, ""
}
