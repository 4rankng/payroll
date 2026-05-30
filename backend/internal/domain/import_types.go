package domain

// ImportError represents a row-level error from an import operation (BCC, etc).
type ImportError struct {
	Row      int    `json:"row"`
	Employee string `json:"employee"`
	Reason   string `json:"reason"`
}
