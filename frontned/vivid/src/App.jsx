import {BrowserRouter as Router, Routes, Route} from 'react-router-dom'
import Home from './pages/Home/Home.jsx'
import About from './pages/About/About.jsx'
import VideoModes from './pages/VideoModes/VideoModes.jsx'
import Navigator from './components/Navigator/Navigator.jsx'
import SingleCamera from './pages/SingleCamera/SingleCamera.jsx'

function App() {

  return (
    <Router>
      <Navigator />
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/about" element={<About />} />
        <Route path="/videomodes" element={<VideoModes />} />
        <Route path="/singlecamera" element={<SingleCamera />} />
      </Routes>
    </Router>
  )
}

export default App
