import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { GitCourseApp } from "./components/GitCourseApp";
import "./styles.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <GitCourseApp />
  </StrictMode>,
);
