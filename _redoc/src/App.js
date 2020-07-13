import React, { useState } from "react";
import { RedocStandalone } from "redoc";
import styles from "./grid.module.css";

async function load() {
  console.log("loading");
  const resp = await fetch("http://localhost:10021/docs/");
  const specs = (await resp.json()).map((r) => ({
    name: r.name,
    url: `http://localhost:10021${r.path}`,
  }));
  console.log(specs);
  return specs;
}

function SidebarButton({ spec, selected, pinned, onClick }) {
  return (
    <button
      style={{
        color: selected ? "red" : "",
      }}
      onClick={onClick}
    >
      {spec.name}
    </button>
  );
}

function Sidebar({ specs, selectedSpec, selectSpec }) {
  return (
    <div className={styles.sidebar}>
      {specs.map((spec) => (
        <SidebarButton
          spec={spec}
          selected={spec === selectedSpec}
          onClick={() => selectSpec(spec)}
        />
      ))}
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
        <div className={styles.content}>
          <RedocStandalone specUrl={selectedSpec.url} />
        </div>
      </div>
    </div>
  );
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
