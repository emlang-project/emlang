// Package config loads the .emlang.yaml configuration file.
//
// Load looks for the file in this order: the path given with -c, the
// EMLANG_CONFIG environment variable, then .emlang.yaml in the current
// directory. A missing file at the default path gives an empty
// configuration, so the CLI works without one. A missing file at an
// explicit path is an error, since the user asked for that file.
package config
