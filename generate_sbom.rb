#!/usr/bin/env ruby
# frozen_string_literal: true

# Generates a software bill of materials (SBOM) for the BDOT Collector.
#
# Usage:
#   ruby generate_sbom.rb [options]
#
# Options:
#   --no-generate   Skip running go-licenses; use an existing NOTICE file
#   --validate      Validate each pkg.go.dev URL via HTTP HEAD (slow ~10min)
#   --package PATH  Go package to scan (default: ./cmd/collector)
#
# Output files:
#   NOTICE   - Raw CSV from go-licenses (name, license_url, license_type)
#   sbom.md  - Human-readable markdown table
#   sbom.csv - Structured CSV suitable for import into spreadsheets

require 'net/http'
require 'uri'
require 'csv'
require 'optparse'

# ---------------------------------------------------------
# Configuration
# ---------------------------------------------------------
DEFAULT_PACKAGE = './cmd/collector'

# Internal module paths to exclude from the report
IGNORE_PACKAGES = %w[
  github.com/observiq/bindplane-otel-collector
].freeze

NOTICE_FILE = 'NOTICE'
OUTPUT_MD   = 'sbom.md'
OUTPUT_CSV  = 'sbom.csv'

# ---------------------------------------------------------
# CLI Options
# ---------------------------------------------------------
options = { validate: false, generate: true, package: DEFAULT_PACKAGE }

OptionParser.new do |opts|
  opts.banner = "Usage: generate_sbom.rb [options]"

  opts.on('--[no-]generate', 'Run go-licenses to generate NOTICE (default: true)') do |v|
    options[:generate] = v
  end

  opts.on('--[no-]validate', 'Validate each pkg.go.dev URL via HTTP (slow, default: false)') do |v|
    options[:validate] = v
  end

  opts.on('--package PATH', "Go package path to scan (default: #{DEFAULT_PACKAGE})") do |v|
    options[:package] = v
  end
end.parse!

# ---------------------------------------------------------
# Helper: run go-licenses and save NOTICE
# ---------------------------------------------------------
def run_go_licenses(package, ignore_packages)
  ignore_flags = ignore_packages.map { |p| "--ignore #{p}" }.join(' ')
  cmd = "go-licenses report #{package} #{ignore_flags} 2>/dev/null"
  puts "Running: #{cmd}"

  output = `#{cmd}`

  unless $CHILD_STATUS.success?
    warn "Warning: go-licenses exited with status #{$CHILD_STATUS.exitstatus} — output may be incomplete"
  end

  if output.empty?
    warn "ERROR: go-licenses produced no output. Is go-licenses installed? (go install github.com/google/go-licenses@latest)"
    exit 1
  end

  output
end

# ---------------------------------------------------------
# Helper: extract version from a license URL
#
# Handles three patterns:
#   v1.2.3       - standard semver tag
#   blob/abc123  - 12-char commit hash
#   +/abcd1234:  - 8-char short hash (diff-style)
# ---------------------------------------------------------
def find_version(input)
  return '' if input.nil? || input.empty?

  if (m = input.match(/v\d+\.\d+\.\d+/))
    m[0]
  elsif (m = input.match(/blob\/\w{12}/))
    m[0][5..16]
  elsif (m = input.match(%r{\+/\w{8}:}))
    m[0][2..9]
  else
    ''
  end
end

# ---------------------------------------------------------
# Helper: validate a URL returns 2xx via HTTP HEAD
# ---------------------------------------------------------
def check_link_validity(url)
  uri = URI.parse(url)
  Net::HTTP.start(uri.host, uri.port, use_ssl: uri.scheme == 'https',
                                      open_timeout: 5, read_timeout: 5) do |http|
    response = http.head(uri.request_uri)
    code = response.code.to_i
    code.between?(200, 299)
  end
rescue StandardError => e
  warn "  URL check failed for #{url}: #{e.message}"
  false
end

# ---------------------------------------------------------
# Step 1: Generate (or load) the NOTICE file
# ---------------------------------------------------------
if options[:generate]
  content = run_go_licenses(options[:package], IGNORE_PACKAGES)
  File.write(NOTICE_FILE, content)
  puts "Wrote #{NOTICE_FILE} (#{content.lines.size} packages)"
end

unless File.exist?(NOTICE_FILE)
  warn "ERROR: #{NOTICE_FILE} not found. Run without --no-generate, or provide the file manually."
  exit 1
end

# ---------------------------------------------------------
# Step 2: Parse NOTICE and build package records
# ---------------------------------------------------------
# Each line: name,license_url,license_type
# e.g. github.com/swaggo/files,https://github.com/swaggo/files/blob/v1.0.1/LICENSE,MIT
#
packages = []
skipped  = []

File.readlines(NOTICE_FILE, chomp: true).each do |line|
  next if line.strip.empty?

  parts = line.split(',')
  next if parts.length < 3

  name            = parts[0].strip
  license_url_raw = parts[1].strip
  license_type    = parts[2..].join(',').strip

  version     = find_version(license_url_raw)
  package_url = "https://pkg.go.dev/#{name}".chomp('/')

  # Build pkg.go.dev release URL.
  # Packages with commit-hash versions get redirected to the base package URL
  # because pkg.go.dev appends a date we don't have (e.g. v0.0.0-20200907-abc123)
  release_url = if version =~ /v\d+\.\d+\.\d+$/
                  version += '+incompatible' if name =~ /sendgrid.+sendgrid-go$/
                  "#{package_url}@#{version}"
                else
                  package_url
                end

  license_pkg_url = "#{release_url}?tab=licenses"

  package = {
    name:        name,
    version:     version,
    license:     license_type,
    package_url: package_url,
    release_url: release_url,
    license_url: license_pkg_url
  }

  if options[:validate]
    print "Validating #{name}... "
    if check_link_validity(package_url) && check_link_validity(release_url) && check_link_validity(license_pkg_url)
      puts 'OK'
      packages << package
    else
      puts 'FAILED (skipping)'
      skipped << package
    end
  else
    packages << package
  end
end

packages.sort_by! { |p| p[:name] }

# ---------------------------------------------------------
# Step 3: Write sbom.md
# ---------------------------------------------------------
File.open(OUTPUT_MD, 'w') do |f|
  f.puts '# Open Source Software Bill of Materials'
  f.puts ''
  f.puts '> Generated by `go-licenses`. Links point to [pkg.go.dev](https://pkg.go.dev).'
  f.puts ''
  f.puts "| Package | Version | License |"
  f.puts "| ------- | ------- | ------- |"
  packages.each do |p|
    name_cell    = "[#{p[:name]}](#{p[:release_url]})"
    version_cell = p[:version].empty? ? '(commit)' : p[:version]
    license_cell = "[#{p[:license]}](#{p[:license_url]})"
    f.puts "| #{name_cell} | #{version_cell} | #{license_cell} |"
  end
end

# ---------------------------------------------------------
# Step 4: Write sbom.csv
# ---------------------------------------------------------
CSV.open(OUTPUT_CSV, 'w') do |csv|
  csv << %w[name version license package_url release_url license_url]
  packages.each { |p| csv << p.values }
end

# ---------------------------------------------------------
# Summary
# ---------------------------------------------------------
puts ''
puts "Done! #{packages.size} packages written to #{OUTPUT_MD} and #{OUTPUT_CSV}"
puts "Skipped #{skipped.size} packages (URL validation failed)" if options[:validate] && skipped.any?
