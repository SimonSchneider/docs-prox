import React, { useState } from "react";
// import { RedocStandalone } from "redoc";
import SwaggerUI from "swagger-ui-react";
import "swagger-ui-react/swagger-ui.css";
import styles from "./grid.module.css";

async function load() {
  console.log("loading");
  const resp = await fetch("/docs/");
  const specs = (await resp.json()).map((r) => ({
    name: r.name,
    url: r.path,
  }));
  console.log(specs);
  return specs;
}

function SidebarButton({ spec, selected, pinned, onClick }) {
  return (
    <div className={styles.sidebarItemWrapper}>
      <div
        className={selected ? styles.selectedSidebarItem : styles.sidebarItem}
        onClick={onClick}
      >
        {spec.name}
      </div>
    </div>
  );
}

function Sidebar({ specs, selectedSpec, selectSpec }) {
  return (
    <div className={styles.sidebarWrapper}>
      <div className={styles.sidebar}>
        {specs.map((spec) => (
          <SidebarButton
            spec={spec}
            selected={spec === selectedSpec}
            onClick={() => selectSpec(spec)}
          />
        ))}
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
