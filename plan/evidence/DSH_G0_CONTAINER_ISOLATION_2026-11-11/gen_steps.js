// usage: node gen_steps.js <slot>  -> steps JSON on stdout
const slot = process.argv[2], all = ['h2-a', 'h3-a', 'h3-b', 'h2-b'];
const others = all.filter(s => s !== slot);
const otherH3 = others.find(s => s.startsWith('h3'));
const W = '-workspace reference/workspace', S = '-subject-kind semantic-data -subject-id sales.order';
const bash = (description, command) => [{ name: 'bash', input: { description, command } }];
const denied = others.map(o => `cat /others/${o}/lane-canary.txt; echo "rc[${o}/lane]=$?"; ls /others/${o}/root; echo "rc[${o}/root]=$?"; cat /others/${o}/root/.agents/skills/yi-tei-modeling/SKILL.md; echo "rc[${o}/skill]=$?"`).join('; ');
console.log(JSON.stringify([
  bash('principal', `id; grep -E '^(Uid|Gid|NoNewPrivs|Seccomp|CapEff)' /proc/self/status; pwd; realpath reference/workspace; sha256sum /opt/tei/tei; readlink -f $(command -v node); node --version; (cd reference/workspace && find . -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum)`),
  bash('tei families', `tei version; echo rc_version=$?; tei commands | wc -c; echo rc_commands=\${PIPESTATUS[0]}; tei inspect ${W} ${S} | wc -c; echo rc_inspect=\${PIPESTATUS[0]}; tei trace ${W} -scope workspace ${S} | head -c 240; echo; echo rc_trace=\${PIPESTATUS[0]}; tei validate ${W} -json | wc -c; echo rc_validate=\${PIPESTATUS[0]}`),
  bash('catalog full', `tei model catalog ${W}`),
  [{ name: 'skill', input: { name: 'yi-tei-modeling' } }],
  bash('own skill', `sha256sum .agents/skills/yi-tei-modeling/SKILL.md .agents/skills/yi-tei-modeling/references/read-only-model-investigation.md; echo rc_own_skill=$?`),
  bash('denied reads', `ls -ld /others/* /reviewer; ${denied}; cat /reviewer/expected.txt; echo "rc[reviewer]=$?"`),
  [{ name: 'read', input: { file_path: '/reviewer/expected.txt' } }],
  [{ name: 'read', input: { file_path: `/others/${otherH3}/root/.agents/skills/yi-tei-modeling/SKILL.md` } }],
  [{ name: 'grep', input: { pattern: 'G0_CANARY', path: '/others' } }],
  bash('other surfaces', `ls /mnt /home 2>&1; cat /proc/net/dev | tail -n +3 | cut -d: -f1; (exec 3<>/dev/tcp/1.1.1.1/80) 2>&1; echo rc_net_ip=$?; getent hosts example.com; echo rc_dns_ext=$?; getent hosts mock-${slot} >/dev/null; echo rc_dns_proxy=$?; touch /trial/x 2>&1; echo rc_w_trial=$?; touch /others/x 2>&1; echo rc_w_others=$?; ls /dsh-home`),
]));
