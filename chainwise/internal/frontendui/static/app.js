const storageKey = "chainwise-bike-profile-v2";

const defaultProfile = {
  bikeName: "My Gravel Bike",
  bikeType: "gravel bike",
  ridingStyle: "commute and weekend rides",
  currentOdometerKm: 1240,
  lastRideDistanceKm: 0,
  lastRideDate: todayISO(),
  lastServiceDate: "2026-04-11",
  lastServiceOdometerKm: 980,
  lastChainLubeOdometerKm: 1160,
  chainCondition: "slightly dry",
  brakeCondition: "good",
  tireCondition: "good"
};

const fields = {
  bikeName: document.getElementById("bike-name"),
  bikeType: document.getElementById("bike-type-input"),
  ridingStyle: document.getElementById("riding-style"),
  currentOdometerKm: document.getElementById("current-odometer"),
  lastRideDate: document.getElementById("last-ride-date"),
  lastServiceDate: document.getElementById("last-service-date"),
  lastServiceOdometerKm: document.getElementById("last-service-odometer"),
  lastChainLubeOdometerKm: document.getElementById("last-chain-lube-odometer"),
  chainCondition: document.getElementById("chain-condition-input"),
  brakeCondition: document.getElementById("brake-condition-input"),
  tireCondition: document.getElementById("tire-condition-input")
};

const checkButton = document.getElementById("check-button");
const addRideButton = document.getElementById("add-ride-button");
const resetButton = document.getElementById("reset-button");

checkButton.addEventListener("click", loadCheck);
addRideButton.addEventListener("click", addRide);
resetButton.addEventListener("click", resetDemoData);

Object.values(fields).forEach((field) => {
  field.addEventListener("change", saveProfileFromForm);
});

loadProfileToForm();
loadCheck();

function addRide() {
  const rideDistanceInput = document.getElementById("ride-distance");
  const rideDistance = numberOrZero(rideDistanceInput.value);

  if (rideDistance <= 0) {
    setStatus("Enter ride km");
    return;
  }

  const profile = getProfileFromForm();
  profile.currentOdometerKm += rideDistance;
  profile.lastRideDistanceKm = rideDistance;
  profile.lastRideDate = todayISO();

  saveProfile(profile);
  setProfileForm(profile);
  rideDistanceInput.value = "";
  setStatus("Ride added");
  loadCheck();
}

async function loadCheck() {
  setLoading(true);

  try {
    const profile = getProfileFromForm();
    saveProfile(profile);

    const params = new URLSearchParams(profile);
    const response = await fetch("/check?" + params.toString());

    if (!response.ok) {
      throw new Error("HTTP " + response.status);
    }

    const data = await response.json();
    renderCheck(data);
  } catch (error) {
    setStatus("Error");
    setText("summary", "Could not calculate the recommendation.");
    setText("reason", "Data loading error: " + error.message);
    document.getElementById("reason").classList.add("error");
    setText("priority", "Error");
  } finally {
    setLoading(false);
  }
}

function renderCheck(data) {
  document.getElementById("reason").classList.remove("error");

  const bike = data.bikeProfile || {};
  const rec = data.recommendation || {};
  const weather = rec.weatherRisk || {};
  const reminder = weather.reminder || {};

  setStatus("Updated");
  setText("title", rec.recommendation || "Maintenance recommendation");
  setText("summary", rec.reason || "Recommendation calculated from your bike data and current weather.");
  setText("priority", readable(rec.priority || "unknown"));

  setText("bike-name-result", bike.name || "—");
  setText("bike-type-result", bike.type || "—");
  setText("odometer-result", valueOrDash(bike.currentOdometerKm, " km"));
  setText("since-lube-result", valueOrDash(rec.kmSinceChainLube, " km"));
  setText("since-service-result", valueOrDash(rec.kmSinceService, " km"));

  setText("city", weather.city || "—");
  setText("condition", readable(weather.condition || "—"));
  setText("weather-risk", readable(weather.risk || "—"));
  setText("weather-source", weather.source || "—");

  setText("reminder-priority", readable(reminder.priority || rec.priority || "—"));
  setText("reminder-type", readable(reminder.type || "—"));
  setText("next-date", reminder.nextDate || rec.nextReminder || "—");
  setText("channel", readable(reminder.channel || "—"));

  setText("reason", rec.reason || weather.reason || "Recommendation calculated.");
}

function getProfileFromForm() {
  const currentOdometerKm = numberOrZero(fields.currentOdometerKm.value);
  let lastServiceOdometerKm = numberOrZero(fields.lastServiceOdometerKm.value);
  let lastChainLubeOdometerKm = numberOrZero(fields.lastChainLubeOdometerKm.value);

  if (lastServiceOdometerKm > currentOdometerKm) {
    lastServiceOdometerKm = currentOdometerKm;
    fields.lastServiceOdometerKm.value = currentOdometerKm;
  }

  if (lastChainLubeOdometerKm > currentOdometerKm) {
    lastChainLubeOdometerKm = currentOdometerKm;
    fields.lastChainLubeOdometerKm.value = currentOdometerKm;
  }

  return {
    bikeName: fields.bikeName.value || defaultProfile.bikeName,
    bikeType: fields.bikeType.value || defaultProfile.bikeType,
    ridingStyle: fields.ridingStyle.value || defaultProfile.ridingStyle,
    currentOdometerKm: currentOdometerKm,
    lastRideDistanceKm: loadProfile().lastRideDistanceKm || 0,
    lastRideDate: fields.lastRideDate.value || todayISO(),
    lastServiceDate: fields.lastServiceDate.value || defaultProfile.lastServiceDate,
    lastServiceOdometerKm: lastServiceOdometerKm,
    lastChainLubeOdometerKm: lastChainLubeOdometerKm,
    chainCondition: fields.chainCondition.value || defaultProfile.chainCondition,
    brakeCondition: fields.brakeCondition.value || defaultProfile.brakeCondition,
    tireCondition: fields.tireCondition.value || defaultProfile.tireCondition
  };
}

function loadProfileToForm() {
  setProfileForm(loadProfile());
}

function setProfileForm(profile) {
  fields.bikeName.value = profile.bikeName;
  fields.bikeType.value = profile.bikeType;
  fields.ridingStyle.value = profile.ridingStyle;
  fields.currentOdometerKm.value = profile.currentOdometerKm;
  fields.lastRideDate.value = profile.lastRideDate;
  fields.lastServiceDate.value = profile.lastServiceDate;
  fields.lastServiceOdometerKm.value = profile.lastServiceOdometerKm;
  fields.lastChainLubeOdometerKm.value = profile.lastChainLubeOdometerKm;
  fields.chainCondition.value = profile.chainCondition;
  fields.brakeCondition.value = profile.brakeCondition;
  fields.tireCondition.value = profile.tireCondition;
}

function saveProfileFromForm() {
  saveProfile(getProfileFromForm());
  setStatus("Saved");
}

function loadProfile() {
  const raw = localStorage.getItem(storageKey);
  if (!raw) {
    return { ...defaultProfile };
  }

  try {
    return { ...defaultProfile, ...JSON.parse(raw) };
  } catch {
    return { ...defaultProfile };
  }
}

function saveProfile(profile) {
  localStorage.setItem(storageKey, JSON.stringify(profile));
}

function resetDemoData() {
  saveProfile({ ...defaultProfile });
  setProfileForm({ ...defaultProfile });
  setStatus("Reset");
  loadCheck();
}

function setLoading(isLoading) {
  checkButton.disabled = isLoading;
  checkButton.textContent = isLoading ? "Checking..." : "Get recommendation";
  if (isLoading) {
    setStatus("Loading");
  }
}

function setStatus(value) {
  setText("status", value);
}

function setText(id, value) {
  document.getElementById(id).textContent = value;
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

function numberOrZero(value) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

function todayISO() {
  return new Date().toISOString().slice(0, 10);
}
