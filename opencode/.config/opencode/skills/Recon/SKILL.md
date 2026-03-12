---
name: "Recon"
description: "Security reconnaissance. USE WHEN recon, reconnaissance, bug bounty, attack surface. SkillSearch('recon') for docs."
---
## Customization

**Before executing, check for user customizations at:**
`~/.config/opencode/skills/PAI/USER/SKILLCUSTOMIZATIONS/Recon/`

If this directory exists, load and apply any PREFERENCES.md, configurations, or resources found there. These override default behavior. If the directory does not exist, proceed with skill defaults.

# recon

**Infrastructure and Network Reconnaissance**

# IP Reconnaissance: 1.2.3.4

## Summary
- IP: 1.2.3.4
- Organization: Example Corp
- ASN: AS12345
- Location: San Francisco, CA, US
- ISP: Example Hosting

## DNS
- Reverse DNS: server.example.com
- Additional PTR: ...

## Network Information
- CIDR: 1.2.3.0/24
- Netblock Owner: Example Corp
- Abuse Contact: abuse@example.com

## Services (Passive)
- Certificates: 3 certificates found
- Historical DNS: ...

## Services (Active - Authorized)
- Open Ports: 22, 80, 443
- Services: SSH (OpenSSH 8.2), HTTP (nginx 1.20.1), HTTPS
- Technologies: nginx, OpenSSH

## Recommendations
- Further investigation areas
- Related assets
- Security observations
```

### Domain Reconnaissance Report
```markdown
# Domain Reconnaissance: example.com

## Summary
- Domain: example.com
- Registrar: Example Registrar
- Registration: 2010-01-15
- Expiry: 2026-01-15

## DNS Records
### A Records
- 1.2.3.4
- 5.6.7.8

### MX Records
- 10 mail.example.com

## Subdomains
- www.example.com (1.2.3.4)
- api.example.com (1.2.3.5)
- admin.example.com (1.2.3.6)

## Email Security
- SPF: Configured
- DMARC: Configured
- DKIM: Configured

## Technologies
- Web Server: nginx
- Framework: React
- CDN: Cloudflare

## Recommendations
- Interesting subdomains: admin.example.com, api.example.com
- Attack surface: 15 web applications identified
```

Reports saved to:
- **Work scratch/** (`~/.config/opencode/MEMORY/WORK/{current_work}/scratch/`) - For iterative artifacts during investigation
- **MEMORY/RESEARCH/** (`~/.config/opencode/MEMORY/RESEARCH/YYYY-MM/`) - For pentest engagements and formal assessments

## MCP Profile Requirements

**For passive recon:** No special MCP profile needed

**For active recon:** Requires security MCP profile
```bash
# Switch to security profile
~/.config/opencode/MCPs/swap-mcp security

# Restart Claude Code to apply
# Then run active reconnaissance
```

Security profile includes:
- httpx (HTTP probing)
- naabu (port scanning)
- Other security tools

## Ethical and Legal Considerations

**CRITICAL RULES:**
1. **Authorization First** - Never perform active scanning without explicit authorization
2. **Scope Validation** - Ensure targets are in-scope for testing
3. **Rate Limiting** - Respectful scanning, no DoS or aggressive techniques
4. **Documentation** - Log all recon activities with timestamps and authorization
5. **Responsible Disclosure** - If vulnerabilities found, follow responsible disclosure
6. **No Destructive Testing** - Reconnaissance only, no exploitation

**Authorization Types:**
- Pentest engagement (written SOW/contract)
- Bug bounty program (in-scope targets only)
- Owned assets (your own infrastructure)
- Research lab/CTF environments

**Never scan:**
- Systems without permission
- Out-of-scope targets
- Critical infrastructure
- Government systems (without specific authorization)

**When in doubt:** Ask for explicit confirmation and documented authorization.

## Usage Examples

### Example 1: Passive Domain Recon
```
User: "Do passive recon on example.com"

Assistant: Activating recon skill with passive-recon workflow.
     Target: example.com
     Mode: Passive only (no active scanning)

     [Performs WHOIS, DNS, cert transparency, IPInfo]
     [Generates report in scratchpad]

     Report: ~/.config/opencode/MEMORY/WORK/{current_work}/scratch/recon-example-com/
```

### Example 2: IP Investigation with Active Scan
```
User: "Investigate IP 1.2.3.4 - I own this server"

Assistant: Activating recon skill with ip-recon workflow.
     Target: 1.2.3.4

     Authorization check: You confirmed this is an owned asset.
     Proceeding with passive + active reconnaissance.

     [Performs IPInfo, reverse DNS, WHOIS, port scan, service detection]
     [Generates comprehensive report]

     Report: Open ports found (22, 80, 443)
             Services: SSH, HTTP, HTTPS
             Technologies: nginx 1.20.1, OpenSSH 8.2
```

### Example 3: Integration with OSINT
```
User: "Do OSINT on Acme Corp and map their infrastructure"

Assistant: Activating OSINT skill...
     Found domains: acme.com, acmecorp.com, acme.io

     Now calling recon for infrastructure mapping...

     [Recon skill maps each domain]
     [Discovers subdomains, IPs, netblocks]
     [Creates comprehensive infrastructure map]

     Report: Complete OSINT + Infrastructure report
             15 domains, 47 subdomains, 3 netblocks identified
```

## Workflow Selection Logic

**Automatic workflow selection based on input:**
- Input matches IP pattern (x.x.x.x) → `IpRecon.md`
- Input matches domain pattern → `DomainRecon.md`
- Input matches CIDR pattern (x.x.x.x/y) → `NetblockRecon.md`
- Input matches ASN pattern (AS####) → `AsnRecon.md`
- User specifies "passive only" → `PassiveRecon.md`

**User can override:**
```
"Use passive-recon workflow on 1.2.3.4"
"Run domain-recon on example.com with active scanning"
```

## Success Criteria

**Passive Recon Success:**
- WHOIS data retrieved
- DNS records enumerated
- Certificate transparency searched
- IPInfo data gathered
- Structured report generated

**Active Recon Success:**
- Authorization confirmed and documented
- Passive recon completed first
- Port scan results (open/closed/filtered)
- Service detection performed
- Banner information gathered
- Technologies identified
- No errors or failures
- Respectful scan timing (no DoS)

## Related Documentation

**Security Skills:**
- `~/.config/opencode/skills/OSINT/` - Entity and people reconnaissance
- `~/.config/opencode/skills/Webassessment/` - Web application testing

**Tool Documentation:**
- IPInfo API: https://ipinfo.io/developers
- Certificate Transparency: https://crt.sh
- WHOIS protocol: RFC 3912

**Best Practices:**
- OWASP Testing Guide: https://owasp.org/www-project-web-security-testing-guide/
- NIST SP 800-115: Technical Guide to Information Security Testing

---

**Remember:** Start passive, confirm authorization before going active, document everything, and be respectful of target systems.
