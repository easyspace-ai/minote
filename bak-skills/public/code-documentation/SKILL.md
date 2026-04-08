---
name: code-documentation
description: Generate comprehensive code documentation including API references, README files, inline comments, and architectural docs. Trigger on queries like "document this code", "generate API docs", "write README", "add comments", or "create documentation".
---

# Code Documentation Skill

## Overview

This skill generates comprehensive code documentation across multiple formats. Use it when the user needs documentation for their codebase, API, or project.

## When to Use This Skill

- User asks to document code or a project
- User wants API reference documentation
- User needs a README file generated
- User wants inline code comments added
- User asks for architectural documentation
- User needs to document a library or framework

## Documentation Types

### 1. README Generation
- Project title and description
- Installation instructions
- Quick start guide
- Usage examples
- Configuration options
- Contributing guidelines
- License information

### 2. API Reference
- Function/method signatures
- Parameter descriptions with types
- Return value documentation
- Usage examples for each endpoint/function
- Error handling documentation
- Authentication requirements

### 3. Inline Code Comments
- Function-level docstrings
- Complex logic explanations
- TODO and FIXME annotations
- Type annotations and hints
- Parameter and return documentation

### 4. Architecture Documentation
- System overview and design
- Component interaction diagrams (in text/mermaid)
- Data flow descriptions
- Technology stack overview
- Directory structure explanation

### 5. Changelog / Release Notes
- Version-based change tracking
- Breaking changes highlighted
- Migration guides
- Deprecation notices

## Best Practices

- **Be concise**: Documentation should be clear and to the point
- **Use examples**: Include code snippets for complex concepts
- **Stay current**: Documentation should match the code
- **Follow conventions**: Use standard formats (JSDoc, GoDoc, Sphinx, etc.)
- **Target audience**: Consider who will read the documentation

## Output Formats

- Markdown (default)
- JSDoc / TSDoc (JavaScript/TypeScript)
- GoDoc (Go)
- Sphinx/RST (Python)
- Javadoc (Java)

## Notes
- Analyze existing code structure before documenting
- Maintain consistency with existing documentation style
- Include both high-level and detailed documentation
- Add cross-references between related sections
