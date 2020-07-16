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

function SidebarButton({ spec }) {
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
        className={`${styles.sidebarItemPin} ${
          spec.pinned ? styles.pinned : ""
        }`}
        onClick={() => {
          spec.togglePin();
        }}
      >
        <Icon disabled={!spec.pinned} name="pin" size="small" />
      </div>
    </div>
  );
}

function Sidebar({ specs }) {
  const [pinnedSpecs, setPinnedSpecs] = useState(getPinned());
  const specsWithPins = specs.map((s) => {
    const isPinned = pinnedSpecs.includes(s.key);
    return {
      ...s,
      pinned: isPinned,
      togglePin: () => {
        isPinned ? removePin(s.key) : addPin(s.key);
        setPinnedSpecs(getPinned());
      },
    };
  });
  const sortedSpecs = [
    ...specsWithPins.filter((s) => s.pinned),
    ...specsWithPins.filter((s) => !s.pinned),
  ];
  const [filter, setFilter] = useState("");
  const filteredSpecs = sortedSpecs.filter((s) =>
    s.name.toLowerCase().includes(filter.toLowerCase())
  );
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
        {filteredSpecs.map((spec) => (
          <SidebarButton spec={spec} />
        ))}
      </div>
      <Route
        exact
        path="/"
        render={() => <Redirect to={`/${sortedSpecs[0].key}`} />}
      />
    </div>
  );
}

function Content({ specs }) {
  return (
    <div className={styles.grid}>
      <Sidebar specs={specs} />
      <div className={styles.contentWrapper}>
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
