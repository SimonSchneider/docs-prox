import React from "react";
import SwaggerUI from "swagger-ui-react";
import "swagger-ui-react/swagger-ui.css";
import styles from "./app.module.css";
import { Route } from "react-router-dom";

function SpecContent({ specs }) {
  return (
    <div className={styles.contentWrapper}>
      <Route
        path="/:key"
        component={({ match }) => {
          const spec = specs.find((s) => s.key === match.params.key);
          return <SwaggerUI url={spec ? spec.url : undefined} />;
        }}
      />
    </div>
  );
}

export default SpecContent;
