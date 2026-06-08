# Delta for backup-diff

## ADDED Requirements

### Requirement: Diff command contract
The system MUST provide `bak diff <id1> <id2>` that loads two backup manifests and categorizes every file as Added, Removed, Modified, or Unchanged.

#### Scenario: All categories present
- GIVEN two backups with overlapping and unique files
- WHEN `bak diff <id1> <id2>` runs
- THEN files are categorized as Added, Removed, Modified, or Unchanged
- AND the command exits 0

#### Scenario: Identical backups
- GIVEN two backups with identical manifests
- WHEN `bak diff <id1> <id2>` runs
- THEN all files are reported as Unchanged
- AND no Added, Removed, or Modified entries appear

#### Scenario: Missing backup
- GIVEN a non-existent first or second backup ID
- WHEN `bak diff <id1> <id2>` runs
- THEN the command exits 1 with an error indicating which backup was not found

### Requirement: Comparison algorithm
The system MUST compare files by canonical SourcePath using the Hash field from each manifest item.

#### Scenario: Cross-platform path normalization
- GIVEN backups created on different OSes with varying path separators
- WHEN `bak diff <id1> <id2>` runs
- THEN `path.Clean(filepath.ToSlash(path))` produces canonical paths for matching
- AND files with equivalent paths are compared correctly

### Requirement: DiffEntry structure
The system MUST represent each difference as a DiffEntry containing SourcePath, Category, and Adapter fields.

#### Scenario: Category assignment — Added
- GIVEN a file exists in backup B but not in backup A
- WHEN `bak diff <idA> <idB>` runs
- THEN the entry's Category is Added

#### Scenario: Category assignment — Modified
- GIVEN a file exists in both backups with different Hash values
- WHEN `bak diff <id1> <id2>` runs
- THEN the entry's Category is Modified

### Requirement: Output format
The system MUST print text-only output grouping files by category.

#### Scenario: Text output
- GIVEN a diff with files in multiple categories
- WHEN `bak diff <id1> <id2>` runs
- THEN output lists files under Added, Removed, Modified, and Unchanged headings

### Requirement: Path traversal prevention
The system MUST reject backup IDs that resolve outside the backup root directory.

#### Scenario: Traversal blocked
- GIVEN a backup ID containing `../`
- WHEN `bak diff <id1> <id2>` runs
- THEN the command exits 1 with a path traversal error
