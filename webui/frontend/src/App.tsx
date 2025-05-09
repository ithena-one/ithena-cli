import './index.css'; // Global styles, Tailwind is via index.html script for now
import LogTable from './features/logs/LogTable'; // Our new main log feature component


function App() {
  return (
    <>
      <LogTable />
    </>
  );
}

export default App;