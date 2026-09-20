async function translateImport() {
  const button = document.getElementById("translate-button");

  if (!button || button.disabled) return;

  const source = window.importEditor.getValue();
  const ruleType = document.getElementById("rule-type-selector").value;

  button.disabled = true;
  button.setAttribute("aria-busy", "true");

  try {
    const response = await fetch("/rules/import/translate", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        source: source,
        type: ruleType,
      }),
    });

    if (!response.ok) {
      throw new Error(`Translation failed: ${response.status}`);
    }

    const translated = await response.text();

    window.translatedEditor.setValue(translated);
  } catch (error) {
    console.error("Failed to translate rule:", error);
  } finally {
    button.disabled = false;
    button.removeAttribute("aria-busy");
  }
}

function initializeEditors() {
  if (!window.monaco) return;

  window.importEditor = window.monaco.editor.create(
    document.getElementById("src"),
    {
      value: "",
      language: "plaintext",
      automaticLayout: true,
    },
  );

  window.translatedEditor = window.monaco.editor.create(
    document.getElementById("result"),
    {
      value: "",
      language: "yaml",
      automaticLayout: true,
    },
  );

  document
    .getElementById("translate-button")
    .addEventListener("click", translateImport);
}

// Monaco may already be loaded.
if (window.monaco) {
  initializeEditors();
} else {
  window.addEventListener("monaco:ready", initializeEditors, {
    once: true,
  });
}
