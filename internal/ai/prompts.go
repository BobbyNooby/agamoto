package ai

const ScanPrompt = `You are a cybersecurity analyst. Given the following nmap scan results, summarize:
1. All open ports and their services
2. Potential security risks based on exposed services
3. Recommended next steps for further investigation

Scan results:
%s`

const ResearchPrompt = `You are a cybersecurity analyst. Given the following web research results about a target, summarize:
1. Known vulnerabilities or breaches
2. Relevant CVEs mentioned
3. Severity assessment

Research results:
%s`

const CombinedPrompt = `You are a cybersecurity analyst. Given the following nmap scan results and web research about a target, provide a comprehensive risk assessment including:
1. Network exposure (open ports, services, versions)
2. Known vulnerabilities and breaches
3. Risk score (Low/Medium/High/Critical)
4. Recommended remediation steps

Scan results:
%s

Research results:
%s`
