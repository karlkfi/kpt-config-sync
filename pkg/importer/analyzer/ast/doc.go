/*
Package ast declares the types used for loading Kubernetes resources from the filesystem into something
like an Abstract Syntax Tree (AST) that allows for writing reusable visitors.  The visitor package
defines some base visitors to use for iterating over the tree and performing transforms.

Each node in the AST implements the "Node" interface which has only one method, "Accept".  For a
visitor to visit a node, it should pass itself to the node's Accept method, and then the node will
call the appropriate "Visit[Type]" method on the visitor.  Iteration is started by having the root
of the tree Accept() the visitor.

Note that this isn't quite exactly a "true" AST as the subtypes are all fully typed, and it is
feasible to iterate over the entire contents in a fully typed manner.  The visitor itself is here
for convenience to make processing and transforming the tree relatively concise.
*/
package ast
