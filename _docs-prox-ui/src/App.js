import React, { useState } from "react";
import { Icon } from "semantic-ui-react";
import SwaggerUI from "swagger-ui-react";
import "swagger-ui-react/swagger-ui.css";
import styles from "./app.module.css";
import { HashRouter, NavLink, Route, Redirect } from "react-router-dom";

function toKey(name) {
  return name.replace(/\s+/g, "-").toLowerCase();
}

async function load() {
  const resp = await fetch("/docs/");
  return (await resp.json()).map((r) => ({
    key: toKey(r.name),
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

function SidebarButton({ spec, pinned, togglePinned }) {
  return (
    <div className={styles.sidebarItemWrapper}>
      <NavLink
        to={`/${spec.key}`}
        activeClassName={styles.selected}
        className={styles.sidebarItem}
      >
        {spec.name}
      </NavLink>
      <div
        className={`${styles.sidebarItemPin} ${pinned ? styles.pinned : ""}`}
        onClick={togglePinned}
      >
        <Icon disabled={!pinned} name="pin" size="small" />
      </div>
    </div>
  );
}

function Sidebar({ specs }) {
  const [pinned, setPinned] = useState(getPinned());
  const [filter, setFilter] = useState("");
  function getTogglePinned(name, isPinned) {
    return () => {
      isPinned ? removePin(name) : addPin(name);
      setPinned(getPinned());
    };
  }
  const allSpecs = specs
    .filter((s) => s.name.toLowerCase().includes(filter.toLowerCase()))
    .map((s) => ({
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
        <div className={styles.sidebarHeader}>
          <div>Docs Prox</div>
          <input
            style={{
              width: "100%",
            }}
            type={"text"}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
        {sortedSpecs.map((spec) => (
          <SidebarButton
            spec={spec}
            pinned={spec.pinned}
            togglePinned={getTogglePinned(spec.name, spec.pinned)}
          />
        ))}
      </div>
    </div>
  );
}

function Content({ specs }) {
  return (
    <div className={styles.grid}>
      <Sidebar specs={specs} />
      <div className={styles.contentWrapper}>
        <Route
          exact
          path="/"
          render={() => <Redirect to={`/${specs[0].key}`} />}
        />
        <Route
          path="/:key"
          component={({ match }) => {
            const spec = specs.find((s) => s.key === match.params.key);
            return <SwaggerUI url={spec ? spec.url : undefined} />;
          }}
        />
      </div>
    </div>
  );
}

function App() {
  const [loaded, setLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [specs, setSpecs] = useState([]);
  if (!loaded) {
    if (!loading) {
      setLoading(true);
      load()
        .then((specs) => setSpecs(specs))
        .then(() => setLoaded(true));
    }
    return <div>Loading content</div>;
  }
  return (
    <HashRouter>
      <div className="App">
        <Content specs={specs} />
      </div>
    </HashRouter>
  );
}

export default App;
