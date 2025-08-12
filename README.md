# storaged

Storaged is a daemon that allows user to self-manage their own assigned quota.

This daemon assumes the following are available:
- munge (typical with installation of Slurm)
- CephFS (for managed directories)
- FreeIPA (for auto-creation of project groups)
- sssd (for doing user and group lookup)

## Directory Structure

We currently assume that there is a root directory containing all the tier
storage. The actual project directory will be created in `<base_dir>/<tier>`
with a symlink from `<base_dir>/<project>` to `<base_dir>/<tier>/<project>`.

## File Permissions

Each user is assigned a quota based on their groups with which they can create
folders. This folder can be shared with other users. This is achieved by
creating a project group (e.g. `project__example`) and adding users into it.

The project folder will have the owner of `owner:project__example` if multiple
users should have access to it. Otherwise, we use `owner:owner` as the file
owner to skip creating groups unless necessary.

While this technically allows the user to chown it to another group, we do not
prohibit the user from doing so as we are still accounting against their quota
as long as the user owner remains them.

## Security

While we have taken measures to do validation whenever applicable and are using
this program in our cluster that is only accessible through the campus network,
this program has not been audited by third parties. By using this program, you
do so at your own risk.
