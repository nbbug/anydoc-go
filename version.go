package anydoc

// Version is the version of the anydoc-go module, bumped in lockstep with the
// pinned anydoc crate version (see Cargo.toml) and the packaged static
// libraries. The build script refuses to build when the two disagree, so a
// document recorded with this version can always be tied back to the exact
// parser it came from.
const Version = "0.2.4"
