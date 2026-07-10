import { AppProviders } from "@/app/providers";
import { DashboardPage } from "@/pages/DashboardPage";

function App() {
  return (
    <AppProviders>
      <DashboardPage />
    </AppProviders>
  );
}

export default App;
