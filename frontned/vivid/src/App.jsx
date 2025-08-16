import {BrowserRouter as Router, Routes, Route} from 'react-router-dom'
import Home from './pages/Home/Home.jsx'
import About from './pages/About/About.jsx'
import VideoModes from './pages/VideoModes/VideoModes.jsx'
import Navigator from './components/Navigator/Navigator.jsx'
import SingleCamera from './pages/SingleCamera/SingleCamera.jsx'
import FourCamera from './pages/FourCamera/FourCamera.jsx'
import CameraStatus from './pages/CameraStatus/CameraStatus.jsx'


function App() {

  return (
    <Router>
      <Navigator />
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/about" element={<About />} />
        <Route path="/videomodes" element={<VideoModes />} />
        <Route path="/singlecamera" element={<SingleCamera />} />
        <Route path="/fourcamera" element={<FourCamera />} />
        <Route path="/camerastatus" element={<CameraStatus />} />

      </Routes>
    </Router>
  )
}

export default App
