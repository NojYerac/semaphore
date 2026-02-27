# Testing Framework Setup for Lightweight Feature-Flag / Rollout Engine

## 1. Go Testing Framework
- Go has a built-in testing framework that can be used for unit tests.
- Create a test file with the suffix `_test.go` to define your tests.

## 2. Example Test
- Here’s a simple example of how to write a test:
  ```go
  package main

  import (
      "testing"
  )

  func TestExample(t *testing.T) {
      expected := 2
      result := 1 + 1
      if result != expected {
          t.Errorf("Expected %d but got %d", expected, result)
      }
  }
  ```

## 3. Running Tests
- Use the following command to run your tests:
  ```bash
  go test
  ```

## 4. Additional Testing Libraries
- Consider using `testify` for more advanced assertions:
  ```bash
  go get github.com/stretchr/testify
  ```

## 5. Documentation
- Document the testing framework setup in this markdown file for future reference.