import {
  Navigate,
  Route,
  Routes,
} from 'react-router-dom'

import AuthLayout from './layouts/AuthLayout'
import ChatLayout from './layouts/ChatLayout'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ProtectedRoute from './components/ProtectedRoute'

export default function App() {
  return (
    <Routes>
      <Route element={<AuthLayout />}>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/register"
          element={<RegisterPage />}
        />
      </Route>

      <Route element={<ProtectedRoute />}>
        <Route path="/app" element={<ChatLayout />} />
      </Route>

      <Route
        path="/"
        element={<Navigate to="/app" replace />}
      />

      <Route
        path="*"
        element={<Navigate to="/app" replace />}
      />
    </Routes>
  )
}