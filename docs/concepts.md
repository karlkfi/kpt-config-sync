## Nomos Concepts

### Namespaces

In Kubernetes, namespaces are the isolation construct for implementing
multi-tenancy and are the parents of all workload resources (pods, replica sets,
service accounts, etc). Namespaces can be used to isolate workloads that have
the same set of human owners as well as to isolate different workload
environments.

Generally anytime you want to have a workload managed by a distinct person or
set of people (e.g. on-call person only for prod workloads, whole development
team for dev workloads), it makes sense to create a new namespace. If you want
to have a common person or set of people be able to perform the same set of
operations within a set of namespaces, create a policyspace.

Nomos with its hierarchical control allows many namespaces to be managed by the
same set of people so it's possible to create more granular namespaces for a
team of people without incurring additional policy administration overhead.

### Policyspaces

With Nomos, we give admins the ability to group namespaces together and to form
groups of groups through a hierarchy. We call a non-leaf node in this tree,
whose leaves are namespaces, a policyspace. We can think of policyspaces as
Organization Units (or Policy Information Point in
[XACML](https://en.wikipedia.org/wiki/XACML#Terminology) parlance). They exist
to delegate policy control to sub organization leaders. This approach has been
long established by LDAP and Active Directory.

Policyspaces can be parents of policyspaces and namespaces. Policyspace and
namespaces must both be globally unique.

### Delegation

As the organization complexity grows, it is important for an admin to have the
ability to delegate administration of a subset of the hierarchy to another
admin. The mechanism for delegation is specific to the source of truth. For
example, on Google Cloud Platform, an admin can grant setIamPolicy permission to
someone who can then set policies independently. In Git, the admin can give
commit permissions to a subtree to someone else.

### Example

One can imagine a brick and mortar retailer having a hierarchy of policyspaces
and namespaces that looks like this

![drawing](img/foo_corp_hierarchy.svg)

In Foo corp, a small team (8-10 people) runs a few microservice workloads that
together provide a bigger component. Everyone on the team has the same set of
roles (e.g. deployment, on-call, coding, etc). We also assume that the small
team will run a dev and staging setup for qualification and will want to ensure
that these environments have different security postures. Also these workloads
need to run in multiple regions.

This gives the ability for the Shipping App Backend team to manage three
different namespaces but only have to maintain one authorization policy for team
members. Each of their namespaces is isolated by environment, allowing
identically-named objects in the three envionments' instantiations of the
backend stack, as well as providing tighter security, e.g. allowing one
namespace to have additional authorized users but not the others, and allocating
private quota to each namespace.
