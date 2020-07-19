import React, { useState } from "react";
import { HashRouter } from "react-router-dom";
import styles from "./app.module.css";
import SpecContent from "./SpecContent";
import Sidebar from "./Sidebar";

async function loadSpecs() {
  const resp = await fetch("/docs/");
  return (await resp.json()).map((r) => ({
    key: r.key,
    name: r.name,
    url: r.path,
  }));
}

function App() {
  const [loaded, setLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [specs, setSpecs] = useState([]);
  if (!loaded) {
    if (!loading) {
      setLoading(true);
      loadSpecs()
        .then((specs) => setSpecs(specs))
        .then(() => setLoaded(true));
    }
    return <div>Loading content</div>;
  }
  return (
    <HashRouter>
      <div className={`App ${styles.grid}`}>
        <Sidebar specs={specs} />
        <SpecContent specs={specs} />
      </div>
    </HashRouter>
  );
}

export default App;
