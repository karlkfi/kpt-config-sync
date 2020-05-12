
# Prow cluster setup

## Follow instructions at prow onboarding

Follow http://go/internal-prow-onboard

Project `stolos-dev` has been set up, so there will be some places that you
will find ACLs that potentially collide with these instructions and you will
likely need to add permission for a new GCP SA.

## Set up perms for "Prow Pod Utilities"

Assign the Roles:

- `Editor`
- `Storage Object Viewer`

For some reason, it fails and complains about not having storage.objects.get,
but adding `Storage Object Viewer` doesn't fix this.  Adding `Editor` fixes, so
it's not clear what perms we need to give it.

## Create "prober-runner" secret

you may need to run make-prober-cred.sh to set up the prober runner.
