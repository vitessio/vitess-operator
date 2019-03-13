package v1alpha2

func (kr *KeyRange) String() string {
	if kr.From != "" || kr.To != "" {
		return kr.From + "-" + kr.To
	}

	// If no From or To is set, then default to the Vitess convention of 0 as they Keyrange string
	return "0"
}
