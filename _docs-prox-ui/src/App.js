import React, { useState } from "react";
import { Icon } from "semantic-ui-react";
import SwaggerUI from "swagger-ui-react";
import "swagger-ui-react/swagger-ui.css";
import styles from "./app.module.css";

async function load() {
  const resp = await fetch("/docs/");
  return (await resp.json()).map((r) => ({
    name: r.name,
    url: r.path,
  }));
}

function getPinned() {
  const pinned = JSON.parse(localStorage.getItem("pinned-items"));
  return pinned ? pinned : [];
}

function setPinned(pinned) {
  localStorage.setItem("pinned-items", JSON.stringify(pinned));
}

function addPin(name) {
  const pinned = getPinned();
  if (pinned.includes(name)) {
    return;
  }
  setPinned([...pinned, name]);
}

function removePin(name) {
  setPinned(getPinned().filter((p) => p !== name));
}

function SidebarButton({ spec, selected, pinned, onClick, togglePinned }) {
  return (
    <div className={styles.sidebarItemWrapper}>
      <div
        className={`${styles.sidebarItem} ${selected ? styles.selected : ""}`}
        onClick={onClick}
      >
        {spec.name}
      </div>
      <div
        className={`${styles.sidebarItemPin} ${pinned ? styles.pinned : ""}`}
        onClick={togglePinned}
      >
        <Icon disabled={!pinned} name="pin" size="small" />
      </div>
    </div>
  );
}

function Sidebar({ specs, selectedSpec, selectSpec }) {
  const [pinned, setPinned] = useState(getPinned());
  function getTogglePinned(name, isPinned) {
    return () => {
      isPinned ? removePin(name) : addPin(name);
      setPinned(getPinned());
    };
  }
  const allSpecs = specs.map((s) => ({
    ...s,
    pinned: pinned.includes(s.name),
  }));
  const sortedSpecs = [
    ...allSpecs.filter((s) => s.pinned),
    ...allSpecs.filter((s) => !s.pinned),
  ];
  return (
    <div className={styles.sidebarWrapper}>
      <div className={styles.sidebar}>
        <div className={styles.sidebarHeader}>Docs Prox</div>
        {sortedSpecs.map((spec) => {
          return (
            <SidebarButton
              spec={spec}
              selected={spec.name === selectedSpec.name}
              onClick={() => selectSpec(spec)}
              pinned={spec.pinned}
              togglePinned={getTogglePinned(spec.name, spec.pinned)}
            />
          );
        })}
      </div>
    </div>
  );
}

function Content({ specs }) {
  const [selectedSpec, selectSpec] = useState(specs[0]);
  return (
    <div className={styles.grid}>
      <Sidebar
        specs={specs}
        selectedSpec={selectedSpec}
        selectSpec={selectSpec}
      />
      <div className={styles.contentWrapper}>
        <SwaggerUI url={selectedSpec.url} />
      </div>
    </div>
  );
  // {/*<RedocStandalone specUrl={selectedSpec.url} />*/}
}

function App() {
  const [loaded, setLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [specs, setSpec] = useState([]);
  if (!loaded) {
    if (!loading) {
      setLoading(true);
      load()
        .then((specs) => setSpec(specs))
        .then(() => setLoaded(true));
    }
    return <div>Loading content</div>;
  }
  return (
    <div className="App">
      <Content specs={specs} />
    </div>
  );
}

export default App;
