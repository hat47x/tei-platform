/* Presentation preferences only. No Host calls, model changes or remote CSS. */
(() => {
  const root = document.documentElement;
  const key = "tei.studio.appearance.v1";
  const themes = ["system", "light", "dark"];
  const densities = ["comfortable", "compact"];
  const system = window.matchMedia("(prefers-color-scheme: dark)");
  let saved;
  try { saved = JSON.parse(localStorage.getItem(key) || "null"); } catch {}
  let theme = themes.includes(saved?.theme) ? saved.theme : "system";
  let density = densities.includes(saved?.density) ? saved.density : "comfortable";
  function apply() {
    root.dataset.teiTheme = theme === "system" ? (system.matches ? "dark" : "light") : theme;
    root.dataset.teiDensity = density;
  }
  apply();
  system.addEventListener("change", () => { if (theme === "system") apply(); });
  document.addEventListener("DOMContentLoaded", () => {
    const themeSelect = document.getElementById("appearance-theme");
    const densitySelect = document.getElementById("appearance-density");
    if (!themeSelect || !densitySelect) return;
    themeSelect.value = theme;
    densitySelect.value = density;
    function change() {
      theme = themes.includes(themeSelect.value) ? themeSelect.value : "system";
      density = densities.includes(densitySelect.value) ? densitySelect.value : "comfortable";
      apply();
      try { localStorage.setItem(key, JSON.stringify({ theme, density })); } catch {}
    }
    themeSelect.addEventListener("change", change);
    densitySelect.addEventListener("change", change);
  });
})();
