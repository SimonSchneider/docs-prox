export function getPinned() {
  const pinned = JSON.parse(localStorage.getItem("pinned-items"));
  return pinned ? pinned : [];
}

function setPinned(pinned) {
  localStorage.setItem("pinned-items", JSON.stringify(pinned));
}

export function addPin(name) {
  const pinned = getPinned();
  if (pinned.includes(name)) {
    return;
  }
  setPinned([...pinned, name]);
}

export function removePin(name) {
  setPinned(getPinned().filter((p) => p !== name));
}
