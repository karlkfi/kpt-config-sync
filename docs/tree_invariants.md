
# Stolos Update Preserving Tree Invariants

Author: briantkennedy

Date: 2018-01-23


# Background

We plan on using git as an option for the source of truth for open source stolos. This means that it's possible for the user to mutate the tree from one valid state to another valid state in an arbitrary manner such that from the set of all possible sequences of mutation operations that will take the tree from state A to state B, there exists a subset of those sequences that would take the tree through an invalid state before reaching state B.


# Summary

It's feasible to avoid this situation if we have the final state of the tree as well as the delta from state A to state B.


# Details


## Terminology



*   Create: an operation that adds a node to the hierarchy
*   Move: an operation that alters the parent relation of a node in a hierarchy. A "move" refers to the node being moved, not the parent of the node being moved.
*   Delete: an operation that removes a node from the hierarchy


## Explanation


### Invariants



1.  All nodes in the hierarchy must have a parent that exists in the hierarchy. The exception is the root node which does not have a parent.
1.  Cycles are not allowed.
1.  There can only be one root node (assuming the root is not renamed)

### Assumptions



1.  We know the current state of the tree on the cluster and it is in a valid state
1.  We know the desired validated state of the tree
1.  Create operations can depend on other create operations
    1.  case: a parent and child created in the same change, the parent must be created first
1.  Update operations can depend on create or move operations
    1.  case: a node is reparented to a node that is created in the same change, the create must proceed first
    1.  case: a node is moved lower in it's own ancestry, eg, (parent to child) a->b->c->d becomes a->c->d->b, the reparent for c must occur before the reparent for b.
1.  Delete operations can depend on other move or delete operations
    1.  case: a parent and child are deleted in the same change, the child must be deleted before the parent


### Summary

There are three possible types of operations we need to be concerned about, create, move (reparent/moving a node) and delete. For ordering moves, if we know the tree will exist in a valid state, at the end of all changes, then we know that any deleted node will not have any children at the end of the state change. Proper ordering entails putting Create, Move and Delete in an order such that the tree never encounters an invalid state. It's possible for a Move or a Create to depend on a node that does not yet exist, or delete to have a child that will be deleted in the future. Given that Moves can depend on Create, we need to process Create operations prior to Move. Given that it's possible for a Delete to depend on an Move, we have to process Moves before deletes. Considering the constraints involved, it's possible for more than one legal ordering of operations to exist for taking the tree from one valid state to another valid state. This algorithm does not intend to deterministically choose which ordering it will use, only that the chosen ordering will result in a legal configuration of the tree after each operation is applied.


### Ordering Create

For create operations, we can topologically sort to find an order that creates parents before children and place these operations first. Since these will be added at their "final" parent for the known good state, we know the subtrees created here will be in a valid state.


### Ordering Update

Next, we have to deal with Update operations. Updates need to be ordered in order to prevent cycles as it's possible to have this occur in a transient state. Since we know the final state of the tree, and we also know that the final state of the tree does not have a cycle, we can ensure we do not ever create a cycle if we order the moves in a manner where the moves are sequenced such that they only get reparented if their new parent and its ancestors are in their final state for the next good configuration. An easy way we can achieve this is to take each move operation, calculate its distance from the root node in the final tree, and sort the moves by distance from the root node. This will ensure that any move applied will always be reparented to the final location for the next good configuration state, and ensure that the tree can never form a cycle because all the parents have already been moved to their final state.


### Ordering Delete

Finally, for delete, we have to perform another topological sort ordered by children first and delete nodes child up.


## Process



1.  Compare the current state of the tree to the desired state of the tree and calculate the set of create, move and delete operations that will transform the tree into the desired state.
1.  Topologically sort all create operations so parent happens before child, add these to the queue in order
1.  Sort all move operations by distance from the final root node and place these in the queue.
1.  Topologically sort all delete operations so child happens before parent, add those to the end of the queue in order.
