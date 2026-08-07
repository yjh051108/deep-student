import { Routes, Route, Navigate } from "react-router-dom";
import { AppShell } from "./layouts/AppShell";
import { ErrorBoundary } from "./layouts/ErrorBoundary";
import { HubPage } from "./pages/HubPage";
import { ChatPage } from "./pages/ChatPage";
import { MindmapPage } from "./pages/MindmapPage";
import { QBankPage } from "./pages/QBankPage";
import { AnkiPage } from "./pages/AnkiPage";
import { ReaderPage } from "./pages/ReaderPage";
import { TranslatePage } from "./pages/TranslatePage";
import { EssayPage } from "./pages/EssayPage";
import { ResearchPage } from "./pages/ResearchPage";
import { PaperPage } from "./pages/PaperPage";
import { MemoryPage } from "./pages/MemoryPage";
import { SkillsPage } from "./pages/SkillsPage";
import { GovernancePage } from "./pages/GovernancePage";
import { SettingsPage } from "./pages/SettingsPage";
import { NotesPage } from "./pages/NotesPage";
import { TodoPage } from "./pages/TodoPage";
import { PomodoroPage } from "./pages/PomodoroPage";
import { LLMUsagePage } from "./pages/LLMUsagePage";
import { SandboxPage } from "./pages/SandboxPage";

export default function App() {
  return (
    <ErrorBoundary>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<Navigate to="/hub" replace />} />
          <Route path="/hub" element={<HubPage />} />
          <Route path="/chat" element={<ChatPage />} />
          <Route path="/mindmap" element={<MindmapPage />} />
          <Route path="/notes" element={<NotesPage />} />
          <Route path="/todo" element={<TodoPage />} />
          <Route path="/pomodoro" element={<PomodoroPage />} />
          <Route path="/qbank" element={<QBankPage />} />
          <Route path="/anki" element={<AnkiPage />} />
          <Route path="/reader" element={<ReaderPage />} />
          <Route path="/translate" element={<TranslatePage />} />
          <Route path="/essay" element={<EssayPage />} />
          <Route path="/research" element={<ResearchPage />} />
          <Route path="/paper" element={<PaperPage />} />
          <Route path="/memory" element={<MemoryPage />} />
          <Route path="/skills" element={<SkillsPage />} />
          <Route path="/llm-usage" element={<LLMUsagePage />} />
          <Route path="/sandbox" element={<SandboxPage />} />
          <Route path="/governance" element={<GovernancePage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<Navigate to="/hub" replace />} />
        </Route>
      </Routes>
    </ErrorBoundary>
  );
}
