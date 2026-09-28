package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/galaxy"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
)

// newGalaxyCmd installs roles and collections from a requirements file
func newGalaxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "galaxy",
		Short: "Install roles and collections (as ansible-galaxy install -r)",
	}
	var (
		file, collectionsPath, rolesPath string
		force                            bool
	)
	install := &cobra.Command{
		Use:   "install -r requirements.yml",
		Short: "Install the roles and collections a requirements file names",
		Long: `Install what onigirazu can use from a requirements.yml:
- collections from git (type: git, git@... or https://...): cloned at their version into
  <collections path>/ansible_collections/<namespace>/<name>, from its galaxy.yml; their roles
  are then found as namespace.collection.role
- roles from git (src) or from Ansible Galaxy (owner.name, cloned from its repository)
Galaxy collections of modules (community.general, ansible.posix, ...) are skipped: onigirazu
has its own modules.

The paths default to the first collections_path and roles_path of onigirazu.yml, the
environment or ansible.cfg (then ~/.ansible/collections and ~/.ansible/roles).`,
		Example: `  onigirazu galaxy install -r requirements.yml
  onigirazu galaxy install -r requirements.yml --collections-path ./collections --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("-r requirements.yml is required")
			}
			req, err := galaxy.Load(file)
			if err != nil {
				return err
			}
			cwd, _ := os.Getwd()
			roles, collections := parser.InstallPaths(cwd)
			if rolesPath == "" {
				rolesPath = roles
			}
			if collectionsPath == "" {
				collectionsPath = collections
			}
			return galaxy.Install(cmd.Context(), req, galaxy.Options{
				RolesPath: rolesPath, CollectionsPath: collectionsPath, Force: force, Log: cmd.OutOrStdout(),
			})
		},
	}
	install.Flags().StringVarP(&file, "role-file", "r", "", "Requirements file")
	install.Flags().StringVar(&collectionsPath, "collections-path", "", "Where collections go")
	install.Flags().StringVar(&rolesPath, "roles-path", "", "Where roles go")
	install.Flags().BoolVar(&force, "force", false, "Install again even when the version is there")
	cmd.AddCommand(install)
	return cmd
}
