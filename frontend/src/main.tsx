import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
// 引入顺序：先 Tailwind 基础，再 shadcn token（必须优先），最后全局重置
import "./styles/global.css";
import "./styles/shadcn-variables.css";

const root = document.getElementById("root");
if (!root) {
  throw new Error("Root element #root not found in index.html");
}

ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>
);
