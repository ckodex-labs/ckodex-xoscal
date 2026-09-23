package tabular

// TabularControl represents a single control row in a spreadsheet or CSV table.
type TabularControl struct {
	ControlID           string `json:"control_id"`
	Component           string `json:"component"`
	Status              string `json:"status"` // "implemented", "planned", "not-applicable"
	ImplementationProse string `json:"implementation_prose"`
	ResponsibleRole     string `json:"responsible_role"`
	Remarks             string `json:"remarks"`
}
