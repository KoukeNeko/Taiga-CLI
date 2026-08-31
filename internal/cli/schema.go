package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) schemaCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "schema [command...]",
		Short: "Print command input/output contracts",
		Args:  cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			values := descriptors()
			if len(args) == 0 {
				items := make([]Descriptor, 0, len(values))
				for _, name := range descriptorNames(values) {
					items = append(items, values[name])
				}
				return a.renderer().List(items, nil)
			}
			name := strings.Join(args, " ")
			descriptor, ok := values[name]
			if !ok {
				return usageError(fmt.Sprintf("unknown command schema %q", name))
			}
			return a.renderer().Data(descriptor)
		},
	}
}
