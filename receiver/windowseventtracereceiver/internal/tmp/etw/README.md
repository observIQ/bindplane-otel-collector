# ETW Package Refactoring

This directory contains a refactored implementation of the Windows Event Tracing (ETW) package. The goal of this refactoring is to organize the code better by directly using the advapi32 and tdh packages, making it more maintainable and easier to understand.

## Goals

1. Port the essential functionality from the original ETW package
2. Use the advapi32 and tdh packages directly for better organization
3. Maintain compatibility with existing receiver code
4. Simplify the implementation where possible

## Components

- `session.go`: Implements the ETW Session mechanism
- `provider.go`: Implements ETW provider functionality
- `consumer.go`: Implements consumption of ETW events
- `event.go`: Defines the Event structure
- `parser.go`: Provides utilities for parsing ETW events

## Usage

The refactored package can be used as a drop-in replacement for the original ETW package in most cases. The API has been kept similar to ensure minimal changes to existing code that uses the package.

## Current Status

This is a work in progress. The following functionality has been implemented:

- Basic ETW session creation and management
- Provider handling
- Consumer for real-time event collection
- Basic event structure

## Next Steps

- Complete callback implementation for event processing
- Add more comprehensive event property parsing
- Improve error handling and reporting
- Add more provider lookup functionality

## Implementation Notes

- Some simplifications have been made compared to the original package
- Error handling has been improved in several places
- Callbacks are implemented differently due to different architecture
- Direct use of advapi32 and tdh packages instead of wrapping them

## Testing

This package should be tested on Windows systems with various ETW providers to ensure compatibility and correct behavior. 