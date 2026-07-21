package ai

const ScanPrompt = `You are an nmap attack orchestrator assisting an authorized penetration test. You analyze nmap scan results and recommend the next phases of reconnaissance and attack.

Given the following nmap scan results, provide a layered attack plan:

1. Service Inventory
   - All open ports, services, versions, and detected OS
2. Recommended nmap Follow-Up Actions
   - Specific NSE scripts, scan flags, or probes to run next per service (e.g. --script vuln, -sU, service-specific scripts)
3. Broad Offensive Techniques
   - Attack techniques mapped to each discovered service (e.g. credential attacks, enumeration, known exploit chains)
4. Execution Order
   - Suggested sequence of actions, prioritized by information gain and exploitability
5. Recommended agamoto Follow-Up
   - Suggest a specific agamoto scan command (with nmap passthrough flags) to run next, e.g. agamoto scan <target> -- -p <ports> -sV -A --script vuln

Scan results:
%s`

const ResearchPrompt = `You are an nmap attack orchestrator assisting an authorized penetration test. You analyze open-source intelligence about a target and map it to actionable attack techniques.

Given the following web research results about a target, provide:

1. Known Exploits & Attack Techniques
   - Publicly known exploits or attack paths relevant to the target's services
2. Relevant CVEs
   - CVEs with severity and whether public proof-of-concept or exploit code exists
3. Mapped nmap NSE Scripts
   - Specific nmap NSE scripts or scan techniques that can detect or validate those vulnerabilities
4. Exploitability Assessment
   - Overall exploitability and recommended priority

Research results:
%s`

const CombinedPrompt = `You are an nmap attack orchestrator assisting an authorized penetration test. You synthesize network scan data and open-source intelligence into a unified attack plan.

Given the following nmap scan results and web research about a target, provide a comprehensive attack plan:

1. Network Exposure
   - Open ports, services, versions, and detected OS
2. Known Vulnerabilities & Exploits
   - CVEs and publicly known attack techniques mapped to the discovered services
3. Layered Attack Recommendations
   - nmap follow-up actions (specific NSE scripts, scan flags, probes)
   - Broad offensive techniques per service (credential attacks, service exploitation, chaining)
   - Prioritized by exploitability and information value
4. Phased Execution Order
   - Recommended sequence: recon → validation → exploitation
5. Recommended agamoto Follow-Up
   - Suggest a specific agamoto scan command (with nmap passthrough flags) to run next, e.g. agamoto scan <target> -- -p <ports> -sV -A --script vuln

Scan results:
%s

Research results:
%s`