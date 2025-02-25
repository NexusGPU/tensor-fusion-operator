<p align="center"><a href="javascript:void(0);" target="_blank" rel="noreferrer"><img width="300" src="https://cdn.tensor-fusion.ai/tensor-fusion.png" alt="Logo"></a></p>

<p align="center">
    <strong><a href="https://tensor-fusion.ai" target="_blank">TensorFusion.AI</a></strong><br/>Next-Generation GPU Virtualization and Pooling for Enterprises<br><b>Less GPUs, More AI Apps.</b>
    <br />
    <a href="https://tensor-fusion.ai/guide/overview"><strong>Explore the docs »</strong></a>
    <br />
    <a href="https://tensor-fusion.ai/guide/overview">View Demo</a>
    |
    <a href="https://github.com/NexusGPU/tensor-fusion/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
    |
    <a href="https://github.com/NexusGPU/tensor-fusion/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
  </p>


# ♾️ Tensor Fusion

[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]
[![LinkedIn][linkedin-shield]][linkedin-url]

Tensor Fusion is a state-of-the-art **GPU virtualization and pooling solution** designed to optimize GPU cluster utilization to its fullest potential.

<a href="https://tensor-fusion.ai/guide/overview"><b>English</b></a>
    |
    <a href="./README-zh.md">简体中文</a>
<br />

## 🌟 Highlights

#### 📐 Fractional GPU with Single TFlops/MiB Precision
#### 🔄 Battle-tested GPU-over-IP Remote GPU Sharing 
#### ⚖️ GPU-first Scheduling and Auto-scaling
#### 📊 Computing Oversubscription and GPU VRAM Expansion
#### 🛫 GPU Live Migration

## 🎬 Demo

WIP

## 🚀 Quick Start

### Onboard Your Own AI Infra

- [Getting Started on Kubernetes](https://tensor-fusion.ai/guide/deployment-k8s)
- [Getting Started on VM](https://tensor-fusion.ai/guide/deployment-vm)
- [Deploy Self-hosted Community Edition](https://tensor-fusion.ai/guide/self-host)

### Try it out

- Explore the demo account: [Demo Console - Working in progress](https://app.tensor-fusion.ai?hint=demo)

- Run following command to try TensorFusion in 3 minutes

```bash
# Step 1: Install TensorFusion in Kubernetes
helm install --repo https://nexusgpu.github.io/tensor-fusion/ --create-namespace

# Step 2. Onboard GPU nodes into TensorFusion cluster
kubectl apply -f https://raw.githubusercontent.com/NexusGPU/tensor-fusion/main/manifests/gpu-node.yaml

# Step 3. Check if cluster and pool is ready
kubectl get gpupools -o wide && kubectl get gpunodes -o wide

# Step3. Create an inference app using virtual, remote GPU resources in TensorFusion cluster
kubectl apply -f https://raw.githubusercontent.com/NexusGPU/tensor-fusion/main/manifests/inference-app.yaml

# Then you can forward the port to test inference, or exec shell
```

(TODO: Asciinema)

### 💬 Discussion

- Discord channel: [https://discord.gg/2bybv9yQNk](https://discord.gg/2bybv9yQNk)
- Discuss anything about TensorFusion: [Github Discussions](https://github.com/NexusGPU/tensor-fusion/discussions)
- Contact us with WeCom for Greater China region: [企业微信](https://work.weixin.qq.com/ca/cawcde42751d9f6a29) 
- Email us: [support@tensor-fusion.com](mailto:support@tensor-fusion.com)
- Schedule [1:1 meeting with TensorFusion founders](https://tensor-fusion.ai/book-demo)


## 🔮 Features & Roadmap

### Core GPU Virtualization Features

- [x] Fractional GPU and flexible oversubscription
- [x] GPU-over-IP, remote GPU sharing with less than 4% performance loss
- [x] GPU VRAM expansion or swap to host RAM
- [ ] None NVIDIA GPU/NPU vendor support

### Pooling & Scheduling & Management

- [x] GPU/NPU pool management in Kubernetes
- [x] GPU-first resource scheduler based on virtual TFlops/VRAM capacity
- [x] GPU-first auto provisioning and bin-packing
- [x] Seamless onboarding experience for Pytorch, TensorFlow, llama.cpp, vLLM, Tensor-RT, SGlang and all popular AI training/serving frameworks
- [x] Basic management console and dashboards
- [ ] Basic autoscaling policies, auto set requests/limits/replicas
- [ ] GPU Group scheduling for LLMs
- [ ] Support different QoS levels

### Enterprise Features

- [x] GPU live-migration, fastest in the world
- [ ] Preloading and P2P distribution of container images, AI models, GPU snapshots etc.
- [ ] Advanced auto-scaling policies, scale to zero, rebalance of hot GPUs
- [ ] Advanced observability features, detailed metrics & tracing/profiling of CUDA calls
- [ ] Multi-tenancy billing based on actual usage
- [ ] Enterprise level high availability and resilience, support topology aware scheduling, GPU node auto failover etc.
- [ ] Enterprise level security, complete on-premise deployment support, encryption in-transit & at-rest
- [ ] Enterprise level compliance, SSO/SAML support, advanced audit, ReBAC control, SOC2 and other compliance reports available

### 🗳️ Platform Support

- [x] Run on Linux Kubernetes clusters
- [x] Run on Linux VMs or Bare Metal (one-click onboarding to Edge K3S)
- [x] Run on Windows (Docs not ready, contact us for support)
- [ ] Run on MacOS (Imagining mount a virtual NVIDIA GPU device on MacOS!)

See the [open issues](https://github.com/NexusGPU/tensor-fusion/issues) for a full list of proposed features (and known issues).

## 🙏 Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".
Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Top contributors

<a href="https://github.com/NexusGPU/tensor-fusion/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=NexusGPU/tensor-fusion" alt="contrib.rocks image" />
</a>

<!-- LICENSE -->
## 🔷 License

1. This repo is open sourced with [Apache 2.0 License](./LICENSE), which includes **GPU pooling, scheduling, management features**, you can use it for free and modify it.
2. **GPU virtualization and GPU-over-IP features** are also free to use as the part of **Community Plan**, the implementation is not fully open sourced
3. Features mentioned in "**Enterprise Features**" above are paid, **licensed users can automatically unlock these features**.

<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/NexusGPU/tensor-fusion.svg?style=for-the-badge
[contributors-url]: https://github.com/NexusGPU/tensor-fusion/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/NexusGPU/tensor-fusion.svg?style=for-the-badge
[forks-url]: https://github.com/NexusGPU/tensor-fusion/network/members
[stars-shield]: https://img.shields.io/github/stars/NexusGPU/tensor-fusion.svg?style=for-the-badge
[stars-url]: https://github.com/NexusGPU/tensor-fusion/stargazers
[issues-shield]: https://img.shields.io/github/issues/NexusGPU/tensor-fusion.svg?style=for-the-badge
[issues-url]: https://github.com/NexusGPU/tensor-fusion/issues
[license-shield]: https://img.shields.io/github/license/NexusGPU/tensor-fusion.svg?style=for-the-badge
[license-url]: https://github.com/NexusGPU/tensor-fusion/blob/master/LICENSE
[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=for-the-badge&logo=linkedin&colorB=555
[linkedin-url]: https://www.linkedin.com/company/tensor-fusion/about

