## Rule Matching

-----

Abstract conditional expressions to perform logical operations between conditions and output matching results.

Encapsulate a simple rule matching mechanism for parameter matching.

Currently supported expressions:

**Currently supported expressions are: ">, >=, <, <=, ==, !=, in, !in, regexp".**

### Configuration Usage

Below is a configuration example in YAML format, written in the router configuration.

```yaml
rule: # Dynamic Rule Matching
  conditions: # Rule conditions
    - key: devid                                    # Condition key
      val: ff3cfb53b1288dc9,ebe564cc9994dddb        # Condition value
      oper: in                                      # Expression evaluation condition, supports >, >=, <, <=, ==, in, !in, !=, regexp
    - key: appver                                   # Condition key
      val: 660                                      # Condition value
      oper: ">="                                    # Expression evaluation condition, supports >, >=, <, <=, ==, in, !in, !=, regexp
  expression: 0&&1                                  # Logical expression, connected by && or ||, using the index of the conditions array
```