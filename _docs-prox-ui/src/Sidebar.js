import React, { useState } from "react";
import { Icon } from "semantic-ui-react";
import styles from "./app.module.css";
import { NavLink, Route, Redirect } from "react-router-dom";
import { getPinned, addPin, removePin } from "./pins";

function SidebarItem({ spec }) {
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
          <SidebarItem spec={spec} />
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

export default Sidebar;
