collector_home="C:/Program Files/observIQ OpenTelemetry Collector"

[
    "#{collector_home}/plugins",
    "#{collector_home}/storage"
].each do |dir|
    describe file(dir) do
        it { should exist }
        it { should be_directory }
    end
end

[
    "#{collector_home}/observiq-otel-collector.exe",
    "#{collector_home}/config.yaml",
    "#{collector_home}/plugins/elasticsearch_logs.yaml",
    "#{collector_home}/plugins/mysql_logs.yaml",
    "#{collector_home}/plugins/redis_logs.yaml"
].each do |file|
    describe file(file) do
        it { should exist }
        it { should be_file }
    end
end

describe service('observiq-otel-collector') do
    it { should be_installed }
    it { should be_enabled }
    it { should be_running }
end

describe registry_key('HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\EventLog\Application\observiq-otel-collector') do
    it { should exist }
    its('EventMessageFile') { should match(/EventCreate\.exe$/) }
    its('TypesSupported') { should cmp 7 }
end
