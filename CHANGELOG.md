# 0.1.10

- Fix generation field not being stripped when using --strip

# 0.1.9

- Fix comma placement when formatting multi-line string values inside lists

# 0.1.8

- Support resources using generateName 
- Add option to remove quotes from object/map keys when not needed
- Match nested interpolation when escaping shell

# 0.1.7

- Escape shell vars in HCL output

# 0.1.6

- Fix crash when trying to use List resources
- Catch panics and print friendly error message

# 0.1.5

- Remove dependency on terraform in go.mod

# 0.1.4

- Fix empty YAML crash (#21)

# 0.1.3

- Ignore empty documents

# 0.1.2

- Add heredoc syntax for multiline strings (#14)
