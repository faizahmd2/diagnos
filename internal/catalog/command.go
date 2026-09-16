package catalog

// ValidateCommand is the last catalog boundary before an LLM-selected
// capability is executed. It intentionally accepts only an exact command
// attached to the selected diagnostic; arbitrary model text never crosses it.
func (c *Catalog) ValidateCommand(capabilityID, command string) bool {
	diagnostic, ok := c.FindCapability(capabilityID)
	if !ok {
		return false
	}
	for _, allowed := range diagnostic.Commands {
		if command == allowed {
			return true
		}
	}
	return false
}
