const STORAGE_KEY = "chainwise.profile.v3";

let rideDateViewDate = null;

const defaultProfile = {
  bikeName: "My Gravel Bike",
  bikeType: "gravel bike",
  ridingStyle: "daily commuting",
  currentOdometerKm: 0,
  lastRideDistanceKm: 0,
  lastRideDate: today(),
  lastServiceDate: today(),

  lastServiceOdometerKm: 0,
  lastChainLubeOdometerKm: 0,
  lastChainReplacementOdometerKm: 0,
  lastBrakeCheckOdometerKm: 0,
  lastTireCheckOdometerKm: 0,

  chainCondition: "unknown",
  chainWear: "unknown",
  brakeCondition: "unknown",
  brakePadThickness: "unknown",
  brakeSymptoms: "none",
  tireCondition: "unknown",
  recentPunctures: 0,
  frontTirePressureBar: "",
  rearTirePressureBar: ""
};

const fields = {
  bikeName: document.getElementById("bike-name"),
  bikeType: document.getElementById("bike-type"),
  ridingStyle: document.getElementById("riding-style"),
  currentOdometerKm: document.getElementById("current-odometer"),
  lastRideDate: document.getElementById("last-ride-date"),
  lastServiceDate: document.getElementById("last-service-date"),
  lastServiceOdometerKm: document.getElementById("last-service-odometer"),
  lastChainLubeOdometerKm: document.getElementById("last-chain-lube-odometer"),
  lastChainReplacementOdometerKm: document.getElementById("last-chain-replacement-odometer"),
  chainCondition: document.getElementById("chain-condition"),
  chainWear: document.getElementById("chain-wear"),
  brakeSymptoms: document.getElementById("brake-symptoms"),
  brakePadThickness: document.getElementById("brake-pad-thickness"),
  tireCondition: document.getElementById("tire-condition"),
  recentPunctures: document.getElementById("recent-punctures"),
  frontTirePressureBar: document.getElementById("front-tire-pressure"),
  rearTirePressureBar: document.getElementById("rear-tire-pressure")
};

const rideDateInput = document.getElementById("ride-date");
const rideDistanceInput = document.getElementById("ride-distance");
const checkButton = document.getElementById("check-button");
const addRideButton = document.getElementById("add-ride-button");
const resetButton = document.getElementById("reset-button");
const forecastList = document.getElementById("forecast-list");
const gearList = document.getElementById("gear-list");

loadProfile();
initCustomSelects();
initRideDatePicker();
loadCheck();

Object.values(fields).forEach((field) => {
  field.addEventListener("input", handleProfileChange);
  field.addEventListener("change", handleProfileChange);
  field.addEventListener("keydown", handleProfileEnter);
});

function handleProfileChange() {
  saveProfile(readProfile());
  syncCustomSelects();
}

function handleProfileEnter(event) {
  if (event.key !== "Enter") {
    return;
  }

  event.preventDefault();
  saveProfile(readProfile());
  loadCheck();
}
checkButton.addEventListener("click", loadCheck);
addRideButton.addEventListener("click", addRide);
resetButton.addEventListener("click", resetDemoData);

rideDistanceInput.addEventListener("keydown", handleRideEnter);
rideDateInput.addEventListener("keydown", handleRideEnter);

function handleRideEnter(event) {
  if (event.key !== "Enter") {
    return;
  }

  event.preventDefault();
  addRide();
}

document.querySelectorAll("[data-action]").forEach((button) => {
  button.addEventListener("click", () => quickAction(button.dataset.action));
});

function loadProfile() {
  const profile = getStoredProfile();
  setFormProfile(profile);
  rideDistanceInput.value = "0";
  rideDateInput.value = today();
}

function getStoredProfile() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return { ...defaultProfile };
    }
    return { ...defaultProfile, ...JSON.parse(raw) };
  } catch (_error) {
    return { ...defaultProfile };
  }
}

function setFormProfile(profile) {
  Object.entries(fields).forEach(([key, field]) => {
    field.value = profile[key] ?? defaultProfile[key] ?? "";
    setText("last-ride-date-display", profile.lastRideDate || "—");
  });
  syncCustomSelects();
}

function readProfile() {
  return {
    bikeName: fields.bikeName.value.trim() || defaultProfile.bikeName,
    bikeType: fields.bikeType.value,
    ridingStyle: fields.ridingStyle.value,
    currentOdometerKm: numberValue(fields.currentOdometerKm.value),
    lastRideDistanceKm: numberValue(getStoredProfile().lastRideDistanceKm || 0),
    lastRideDate: fields.lastRideDate.value || today(),
    lastServiceDate: fields.lastServiceDate.value || daysAgo(18),
    lastServiceOdometerKm: numberValue(fields.lastServiceOdometerKm.value),
    lastChainLubeOdometerKm: numberValue(fields.lastChainLubeOdometerKm.value),
    lastChainReplacementOdometerKm: numberValue(fields.lastChainReplacementOdometerKm.value),
    lastBrakeCheckOdometerKm: numberValue(localStorage.getItem("chainwise.lastBrakeCheckOdometerKm") || 0),
    lastTireCheckOdometerKm: numberValue(localStorage.getItem("chainwise.lastTireCheckOdometerKm") || 0),
    chainCondition: fields.chainCondition.value,
    chainWear: fields.chainWear.value,
    brakeCondition: "unknown",
    brakePadThickness: fields.brakePadThickness.value,
    brakeSymptoms: fields.brakeSymptoms.value,
    tireCondition: fields.tireCondition.value,
    recentPunctures: numberValue(fields.recentPunctures.value),
    frontTirePressureBar: fields.frontTirePressureBar.value,
    rearTirePressureBar: fields.rearTirePressureBar.value
  };
}

function saveProfile(profile) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(profile));
}

function addRide() {
  const distance = numberValue(rideDistanceInput.value);
  if (distance <= 0) {
    setText("status", "Ride needed");
    return;
  }

  const profile = readProfile();
  profile.currentOdometerKm += distance;
  profile.lastRideDistanceKm = distance;
  profile.lastRideDate = rideDateInput.value || today();
  saveProfile(profile);
  setFormProfile(profile);
  rideDateInput.value = today();
  syncRideDatePicker();
  loadCheck();
}

function quickAction(action) {
  const profile = readProfile();
  const current = profile.currentOdometerKm;

  if (action === "chain-lubed") {
    profile.lastChainLubeOdometerKm = current;
    profile.chainCondition = "good";
  }

  if (action === "chain-replaced") {
    profile.lastChainReplacementOdometerKm = current;
    profile.lastChainLubeOdometerKm = current;
    profile.chainCondition = "good";
    profile.chainWear = "below 0.5%";
  }

  if (action === "service-done") {
    profile.lastServiceOdometerKm = current;
    profile.lastServiceDate = today();
    localStorage.setItem("chainwise.lastBrakeCheckOdometerKm", String(current));
    localStorage.setItem("chainwise.lastTireCheckOdometerKm", String(current));
  }

  if (action === "brakes-checked") {
    localStorage.setItem("chainwise.lastBrakeCheckOdometerKm", String(current));
    profile.brakeSymptoms = "none";
    profile.brakePadThickness = "unknown";
  }

  if (action === "tires-checked") {
    localStorage.setItem("chainwise.lastTireCheckOdometerKm", String(current));
    profile.tireCondition = "good";
    profile.recentPunctures = 0;
  }

  saveProfile(profile);
  setFormProfile(profile);
  loadCheck();
}

function resetDemoData() {
  localStorage.removeItem(STORAGE_KEY);
  localStorage.removeItem("chainwise.lastBrakeCheckOdometerKm");
  localStorage.removeItem("chainwise.lastTireCheckOdometerKm");
  setFormProfile({ ...defaultProfile });
  rideDistanceInput.value = "32";
  loadCheck();
}

async function loadCheck() {
  setLoading(true);

  try {
    const profile = readProfile();
    saveProfile(profile);

    const response = await fetch("/check?" + toQuery(profile));
    if (!response.ok) {
      throw new Error("HTTP " + response.status);
    }

    const data = await response.json();
    renderCheck(data);
  } catch (error) {
    setText("status", "Error");
    setText("summary", "Could not load recommendation.");
    setText("reason", "Error loading data: " + error.message);
    document.getElementById("reason").classList.add("error");
    setPriority("error");
  } finally {
    setLoading(false);
  }
}

function renderCheck(data) {
  const bike = data.bikeProfile || {};
  const rec = data.recommendation || {};
  const weather = rec.weatherRisk || {};
  const rideAdvice = weather.rideAdvice || {};

  setText("status", "Updated");
  setText("title", rec.recommendation || "Maintenance recommendation");
  setText("summary", rec.reason || "Recommendation calculated from odometer, weather and optional checks.");
  setPriority(rec.priority || "waiting");

  setText("result-bike-name", bike.name || "—");
  setText("result-bike-type", bike.type || "—");
  setText("result-odometer", valueOrDash(bike.currentOdometerKm, " km"));
  setText("result-chain-km", valueOrDash(rec.kmSinceChainLube, " km"));
  setText("result-service-km", valueOrDash(rec.kmSinceService, " km"));

  setText("city", weather.city || "—");
  setText("condition", readable(weather.condition || "—"));
  setText("feels-like", tempValue(weather.apparentTemperatureC));
  setText("wind-speed", windValue(weather.windSpeedMs));
  setText("wind-gusts", windValue(weather.windGustsMs));

  setText("ride-title", rideAdvice.title || "—");
  setText("ride-message", rideAdvice.message || "—");
  renderChips(gearList, rideAdvice.gear || []);
  renderForecast(rec.componentForecast || []);
}

function renderForecast(items) {
  forecastList.innerHTML = "";

  if (!items.length) {
    forecastList.innerHTML = '<div class="forecast-item"><div>No forecast available yet.</div></div>';
    return;
  }

  items.forEach((item) => {
    const el = document.createElement("article");
    el.className = "forecast-item";

    const kmText = item.remainingKm > 0
      ? item.remainingKm + " km left"
      : item.overdueKm > 0
        ? item.overdueKm + " km overdue"
        : "due now";

    el.innerHTML = `
      <div class="forecast-label">${escapeHtml(item.label || item.component)}</div>
      <div class="forecast-status status-${escapeHtml(item.status || "ok")}">${readable(item.status || "ok")}</div>
      <div class="forecast-reason">${escapeHtml(item.reason || item.action || "")}</div>
      <div class="forecast-km">${escapeHtml(kmText)}</div>
    `;

    forecastList.appendChild(el);
  });
}

function renderChips(container, values) {
  container.innerHTML = "";
  values.forEach((value) => {
    const chip = document.createElement("span");
    chip.className = "chip";
    chip.textContent = readable(value);
    container.appendChild(chip);
  });
}

function setLoading(isLoading) {
  checkButton.disabled = isLoading;
  checkButton.textContent = isLoading ? "Checking..." : "Get recommendation";
  setText("status", isLoading ? "Loading" : "Ready");
}

function setPriority(priority) {
  const el = document.getElementById("priority");
  el.textContent = readable(priority);
  el.className = "priority-badge " + String(priority).toLowerCase();
}

function toQuery(profile) {
  const params = new URLSearchParams();
  Object.entries(profile).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      params.set(key, value);
    }
  });
  return params.toString();
}

function setText(id, value) {
  const element = document.getElementById(id);
  if (!element) {
    return;
  }
  element.textContent = value;
}

function readable(value) {
  return String(value).replaceAll("_", " ");
}

function valueOrDash(value, suffix) {
  if (value === undefined || value === null || value === "") {
    return "—";
  }
  return String(value) + suffix;
}

function tempValue(value) {
  if (value === undefined || value === null || Number.isNaN(Number(value))) {
    return "—";
  }
  return Number(value).toFixed(1) + " °C";
}

function windValue(value) {
  if (value === undefined || value === null || Number.isNaN(Number(value))) {
    return "—";
  }
  return Number(value).toFixed(1) + " m/s";
}

function numberValue(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }
  return Math.round(parsed);
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

function daysAgo(days) {
  const date = new Date();
  date.setDate(date.getDate() - days);
  return date.toISOString().slice(0, 10);
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function initCustomSelects() {
  document.querySelectorAll("select").forEach((select) => {
    if (select.dataset.customSelect === "true") {
      return;
    }

    select.dataset.customSelect = "true";
    select.classList.add("native-select-hidden");

    const wrapper = document.createElement("div");
    wrapper.className = "cw-select";
    wrapper.dataset.selectId = select.id;

    const trigger = document.createElement("button");
    trigger.type = "button";
    trigger.className = "cw-select-trigger";
    trigger.setAttribute("aria-haspopup", "listbox");
    trigger.setAttribute("aria-expanded", "false");

    const menu = document.createElement("div");
    menu.className = "cw-select-menu";
    menu.setAttribute("role", "listbox");

    Array.from(select.options).forEach((option) => {
      const item = document.createElement("button");
      item.type = "button";
      item.className = "cw-select-option";
      item.textContent = option.textContent;
      item.dataset.value = option.value;
      item.setAttribute("role", "option");

      item.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopPropagation();

        select.value = option.value;
        select.dispatchEvent(new Event("change", { bubbles: true }));

        updateCustomSelect(select);
        closeCustomSelects();
      });

      menu.appendChild(item);
    });

    trigger.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();

      const wasOpen = wrapper.classList.contains("is-open");
      closeCustomSelects();

      if (!wasOpen) {
        wrapper.classList.add("is-open");
        trigger.setAttribute("aria-expanded", "true");
      }
    });

    wrapper.appendChild(trigger);
    wrapper.appendChild(menu);

    select.insertAdjacentElement("afterend", wrapper);
    updateCustomSelect(select);
  });

  if (!window.chainwiseCustomSelectListenerInstalled) {
    window.chainwiseCustomSelectListenerInstalled = true;

    document.addEventListener("click", () => {
      closeCustomSelects();
    });

    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        closeCustomSelects();
      }
    });
  }
}

function updateCustomSelect(select) {
  const wrapper = document.querySelector(`.cw-select[data-select-id="${select.id}"]`);

  if (!wrapper) {
    return;
  }

  const trigger = wrapper.querySelector(".cw-select-trigger");
  const options = wrapper.querySelectorAll(".cw-select-option");
  const selectedOption = Array.from(select.options).find((option) => option.value === select.value);

  trigger.textContent = selectedOption ? selectedOption.textContent : "Select";

  options.forEach((option) => {
    const isActive = option.dataset.value === select.value;
    option.classList.toggle("active", isActive);
    option.setAttribute("aria-selected", String(isActive));
  });
}

function syncCustomSelects() {
  document.querySelectorAll("select").forEach((select) => {
    updateCustomSelect(select);
  });
}

function closeCustomSelects() {
  document.querySelectorAll(".cw-select").forEach((select) => {
    select.classList.remove("is-open");

    const trigger = select.querySelector(".cw-select-trigger");
    if (trigger) {
      trigger.setAttribute("aria-expanded", "false");
    }
  });
}

function initRideDatePicker() {
  const picker = document.getElementById("ride-date-picker");
  const trigger = document.getElementById("ride-date-trigger");
  const prev = document.getElementById("ride-date-prev");
  const next = document.getElementById("ride-date-next");
  const todayButton = document.getElementById("ride-date-today");

  if (!picker || !trigger || !rideDateInput) {
    return;
  }

  if (!rideDateInput.value) {
    rideDateInput.value = today();
  }

  rideDateViewDate = parseISODate(rideDateInput.value) || new Date();

  trigger.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();

    const isOpen = picker.classList.contains("is-open");
    closeRideDatePicker();

    if (!isOpen) {
      picker.classList.add("is-open");
      renderRideDatePicker();
    }
  });

  prev.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();

    rideDateViewDate.setMonth(rideDateViewDate.getMonth() - 1);
    renderRideDatePicker();
  });

  next.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();

    rideDateViewDate.setMonth(rideDateViewDate.getMonth() + 1);
    renderRideDatePicker();
  });

  todayButton.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();

    setRideDate(today());
    closeRideDatePicker();
  });

  document.addEventListener("click", () => {
    closeRideDatePicker();
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeRideDatePicker();
    }
  });

  syncRideDatePicker();
}

function renderRideDatePicker() {
  const monthLabel = document.getElementById("ride-date-month");
  const grid = document.getElementById("ride-date-grid");

  if (!monthLabel || !grid || !rideDateViewDate) {
    return;
  }

  const year = rideDateViewDate.getFullYear();
  const month = rideDateViewDate.getMonth();

  monthLabel.textContent = rideDateViewDate.toLocaleDateString("en-US", {
    month: "long",
    year: "numeric"
  });

  grid.innerHTML = "";

  const firstDay = new Date(year, month, 1);
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const leadingEmptyDays = (firstDay.getDay() + 6) % 7;
  const selected = rideDateInput.value;

  for (let i = 0; i < leadingEmptyDays; i++) {
    const empty = document.createElement("span");
    empty.className = "cw-date-empty";
    grid.appendChild(empty);
  }

  for (let day = 1; day <= daysInMonth; day++) {
    const date = new Date(year, month, day);
    const value = formatISODate(date);

    const button = document.createElement("button");
    button.type = "button";
    button.className = "cw-date-day";
    button.textContent = String(day);

    if (value === selected) {
      button.classList.add("active");
    }

    button.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();

      setRideDate(value);
      closeRideDatePicker();
    });

    grid.appendChild(button);
  }
}

function setRideDate(value) {
  rideDateInput.value = value;
  rideDateInput.dispatchEvent(new Event("change", { bubbles: true }));
  syncRideDatePicker();
}

function syncRideDatePicker() {
  const trigger = document.getElementById("ride-date-trigger");

  if (!trigger || !rideDateInput) {
    return;
  }

  if (!rideDateInput.value) {
    rideDateInput.value = today();
  }

  trigger.textContent = rideDateInput.value;
  rideDateViewDate = parseISODate(rideDateInput.value) || new Date();

  renderRideDatePicker();
}

function closeRideDatePicker() {
  const picker = document.getElementById("ride-date-picker");

  if (picker) {
    picker.classList.remove("is-open");
  }
}

function parseISODate(value) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);

  if (!match) {
    return null;
  }

  return new Date(
    Number(match[1]),
    Number(match[2]) - 1,
    Number(match[3])
  );
}

function formatISODate(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
}