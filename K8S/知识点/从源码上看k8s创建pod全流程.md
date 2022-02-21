# 从源码上看k8s创建pod全流程

***<u>以下代码均是基于K8s正式发布版V1.20.2进行分析。</u>***



## 整体流程图

从整体具体的上看，kubectl负责接收用户的输入，做初步的处理后，按照kube-apiserver的处理要求生成具体的请求；kube-apiserver是所有请求的入口，它负责所有请求的通用检查和分发以及对外提供资源状态的查询等等；kubelet负责具体pod的生命周期管理和节点整体状态数据的上报。

Kubernetes是基于事件+控制器模式实现的，因此在代码中并没有一个贯穿pod创建控制流程始终的代码存在。涉及到的组件更像是团队接力赛选手，大家在完成自己的工作后，就把棒交出去了(更新信息/触发某种事件)，下一个选手根据自己感兴趣的事件去选择要做的事情(例如watch监听)，并努力把自己分内的事做好，然后再交出去(更新信息/触发某种事件)。

# kubectl

客户端参数检查验证和对象生成

代码入口：

vendor/k8s.io/kubectl/pkg/cmd/run/run.go#L246

## 参数检查与验证

主要包含镜像名称校验，镜像拉取策略校验等

```go
// vendor/k8s.io/kubectl/pkg/cmd/run/run.go#L276 
imageName := o.Image 
if imageName == "" { 
  return fmt.Errorf("--image is required") 
} 
validImageRef := reference.ReferenceRegexp.MatchString(imageName)
if !validImageRef { 
  return fmt.Errorf("Invalid image name %q: %v", imageName, reference.ErrReferenceInvalidFormat)   
}

// vendor/k8s.io/kubectl/pkg/cmd/run/run.go#L310 
if err := verifyImagePullPolicy(cmd); err != nil {
  return err
}
```

## 对象生成

获取pod默认生成器

```go
// vendor/k8s.io/kubectl/pkg/cmd/run/run.go#L314 
generators := generateversioned.GeneratorFn("run") 
// 加载run下说有的生成器，目前只剩下一个pod的生成器，历史版本上还有job、deployment等等，参考kubectl run命令历史小节 
generator, found := generators[generateversioned.RunPodV1GeneratorName] // "run-pod/v1" 
if !found { 
  return cmdutil.UsageErrorf(cmd, "generator %q not found", o.Generator)   
} 
// vendor/k8s.io/kubectl/pkg/generate/versioned/generator.go#L94 
case "run": // run子命令下注册的默认生成器 
	generator = map[string]generate.Generator{ RunPodV1GeneratorName: BasicPod{},     
```

生成运行时对象

```go
// vendor/k8s.io/kubectl/pkg/cmd/run/run.go#L330 
var createdObjects = []*RunObject{} 
runObject, err := o.createGeneratedObject(f, cmd, generator, names, params, cmdutil.GetFlagString(cmd, "overrides")) // 这里开始发起创建运行时对象 
if err != nil { 
  return err 
} 
createdObjects = append(createdObjects, runObject) 
// vendor/k8s.io/kubectl/pkg/cmd/run/run.go#L616 
func (o *RunOptions) createGeneratedObject(f cmdutil.Factory, cmd *cobra.Command, generator generate.Generator, names []generate.GeneratorParam, params map[string]interface{}, overrides string) (*RunObject, error) { 
  // 验证生成器参数  
  err := generate.ValidateParams(names, params)   
  // 生成器生成对象  obj, err := generator.Generate(params)  
  // API分组和版本协调  
  mapper, err := f.ToRESTMapper() 
  // run has compiled knowledge of the thing is creating 
  gvks, _, err := scheme.Scheme.ObjectKinds(obj) 
  mapping, err := mapper.RESTMapping(gvks[0].GroupKind(), gvks[0].Version)  
  if o.DryRunStrategy != cmdutil.DryRunClient {  
    // 客户端实例构建 
    client, err := f.ClientForMapping(mapping) 
    // 具体实例取决于f是怎么实例化的 // 发送HTTP请求   
    actualObj, err = resource. NewHelper(client, mapping). DryRun(o.DryRunStrategy == cmdutil.DryRunServer). // 动态配置server side dry run 
    WithFieldManager(o.fieldManager). // 更新管理者   
    Create(o.Namespace, false, obj)  
  } 
}
```

关于API groups和version发现与协商

Kubernetes使用的API是带版本号并且被分成了API groups。一个API group是指一组操作资源类似的API集合。Kubernetes一般支持多版本的API groups，kubectl为了找到最合适的API，需要只通过发现机制来获取kube-api暴露的schema文档(通过OpenAPI格式）。为了提高性能，一般kubectl会在本地~/.kube/cache/discovery目录缓存这些schema文件。

## 处理返回结果

得到api-server返回值后，进行后续处理。按照正确的格式输出创建的对象。

```go
// vendor/k8s.io/kubectl/pkg/cmd/run/run.go#L430 
if runObject != nil { 
	if err := o.PrintObj(runObject.Object); err != nil { 
		return err 
	}
}
```

客户端认证支持

为了确保请求发送成功，kubectl需要具备认证的能力。用户凭证几乎总是存储在本地磁盘的kubeconfig文件中。为了定位该文件，kubectl会按照以下步骤加载该文件

1. 如果--kubeconfig指定了文件，则使用这个文件
2. 如果$KUBECONFIG环境变量定义了，则使用该环境变量指向的文件
3. 在本地home目录下，例如~/.kube，搜索并使用第一个找到的文件

在文件解析完成后，kubectl就可以确定当前使用的上下文，指向的集群以及当前用户关联的认证信息。

# kube-apiserver

## 认证

kube-apiserver是客户端与系统组件之间主要的持久化和查询集群状态界面。首要的kube-apiserver需要知道请求的发起者是谁。

apiserver如何对请求做认证？当服务第一次启动时，它会查看用户提供的所有命令行参数，然后组装成一个合适的认证器列表。每个请求到来后都要逐个通过认证器的检查，直到有一个认证通过。

```go
// vendor/k8s.io/apiserver/pkg/authentication/request/union/union.go#L53 
// AuthenticateRequest authenticates the request using a chain of authenticator.Request objects. 
func (authHandler *unionAuthRequestHandler) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) { 
  var errlist []error 
  for _, currAuthRequestHandler := range authHandler.Handlers { 
    resp, ok, err := currAuthRequestHandler.AuthenticateRequest(req) 
    if err != nil { 
      if authHandler.FailOnError {
        return resp, ok, err
      }
      errlist = append(errlist, err) continue
    }
    if ok {
      return resp, ok, err
    }
  }
  return nil, false, utilerrors.NewAggregate(errlist)
}
```

认证器的初始化流程

```go
// pkg/kubeapiserver/authenticator/config.go#L95
// New returns an authenticator.Request or an error that supports the standard
// Kubernetes authentication mechanisms.
func (config Config) New() (authenticator.Request, *spec.SecurityDefinitions, error) {
  var authenticators []authenticator.Request
  authenticator := union.New(authenticators...)
  // 各种初始化
  authenticator = group.NewAuthenticatedGroupAdder(authenticator)
  if config.Anonymous {
    // If the authenticator chain returns an error, return an error (don't consider a bad bearer token
    // or invalid username/password combination anonymous).
    authenticator = union.NewFailOnError(authenticator, anonymous.NewAuthenticator())
  }
  return authenticator, &securityDefinitions, nil
}
```

如下图所示，假设所有的认证器都被启用，当客户端发送请求到kube-apiserver服务，该请求会进入Authentication Handler函数（处理认证相关的Handler函数），在Authentication Handler函数中，会遍历已启用的认证器列表，尝试执行每个认证器，当有一个认证器返回true时，则认证成功，否则继续尝试下一个认证器。

当所有认证器都认证失败后，请求将会被拒绝，合并后的错误会返回给客户端。如果认证成功，Authorization头信息将会从请求中移除，用户信息会添加到请求的上下文信息中。这样后续步骤就可以访问到认证阶段确定的请求用户的信息了。

## 鉴权

虽然现在kube-apiserver已经成功地验证了请求者的身份信息，但是在进行下一步之前还得确保请求者是否有权限去操作。身份认证和鉴权不是同一个事情，要想进一步使用，kube-apiserver需要对我们进行鉴权。

类似认证器的处理方法，kube-apiserver需要基于用户提供的命令行参数，来组装一个合适的鉴权器列表来处理每一个请求。当所有的鉴权器都拒绝该请求时，请求会终止，并且请求方会得到Forbidden的答复。如果任何一个鉴权器批准了请求，那么请求鉴权成功，将会进入下一阶段处理。

鉴权器初始化

kube-apiserver目前提供了6种授权机制，分别是AlwaysAllow、AlwaysDeny、ABAC、Webhook、RBAC、Node，可通过指定--authorization-mode参数设置授权机制，至少需要指定一个。

```go
// pkg/kubeapiserver/authorizer/config.go#L71
// New returns the right sort of union of multiple authorizer.Authorizer objects // based on the authorizationMode or an error.
func (config Config) New() (authorizer.Authorizer, authorizer.RuleResolver, error) {
  if len(config.AuthorizationModes) == 0 {
    return nil, nil, fmt.Errorf("at least one authorization mode must be passed")
  }
}
```

鉴权器决策状态

```go
// vendor/k8s.io/apiserver/pkg/authorization/authorizer/interfaces.go#L149
type Decision int const
( 
  // DecisionDeny means that an authorizer decided to deny the action.
  DecisionDeny
  Decision = iota
  // DecisionAllow means that an authorizer decided to allow the action.
  DecisionAllow
  // DecisionNoOpionion means that an authorizer has no opinion on whether
  // to allow or deny an action.
  DecisionNoOpinion
)
```

当决策状态是DecisionDeny或DecisionNoOpinion时会交由下一个鉴权器继续处理，如果没有下一个鉴权器则鉴权失败。当决策状态是DecisionAllow时鉴权成功，请求被接受。

## Admission control

在认证和授权之后，对象被持久化之前，拦截kube-apiserver的请求，对请求的资源对象进行自定义操作（校验、修改或者拒绝请求）。为什么需要有这一个环节？为了集群的稳定性，在资源对象被正式接纳前，需要由系统内其他组件对待创建的资源先进行一系列的检查，确保符合整个集群的预期和规则，从而防患于未然。这是在etcd创建资源前的最后一道保障。

插件实现接口

```go
// vendor/k8s.io/apiserver/pkg/admission/interfaces.go#L123
// Interface is an abstract, pluggable interface for Admission Control decisions.
type Interface interface {
	// Handles returns true if this admission controller can handle the given operation
	// where operation can be one of CREATE, UPDATE, DELETE, or CONNECT
	Handles(operation Operation) bool
}

type MutationInterface interface {
	Interface

	// Admit makes an admission decision based on the request attributes.
	// Context is used only for timeout/deadline/cancellation and tracing information.
	Admit(ctx context.Context, a Attributes, o ObjectInterfaces) (err error)
}

// ValidationInterface is an abstract, pluggable interface for Admission Control decisions.
type ValidationInterface interface {
	Interface

	// Validate makes an admission decision based on the request attributes.  It is NOT allowed to mutate
	// Context is used only for timeout/deadline/cancellation and tracing information.
	Validate(ctx context.Context, a Attributes, o ObjectInterfaces) (err error)
}
```

两种准入控制器

1. 变更准入控制器（Mutating Admission Controller）用于变更信息，能够修改用户提交的资源对象信息。
2. 验证准入控制器（Validating Admission Controller）用于身份验证，能够验证用户提交的资源对象信息。

变更准入控制器运行在验证准入控制器之前。

准入控制器的运行方式与认证和鉴权方式类似，不同的地方是任何一个准入控制器失败后，整个准入控制流程就会结束，请求失败。

准入控制器以插件的形式运行在kube-apiserver进程中，插件化的好处在于可扩展插件并单独启用/禁用指定插件，也可以将每个准入控制器称为准入控制器插件。

客户端发起一个请求，在请求经过准入控制器列表时，只要有一个准入控制器拒绝了该请求，则整个请求被拒绝（HTTP 403Forbidden）并返回一个错误给客户端。

## ETCD存储

经过身份认证、鉴权以及准入控制检查后，kube-apiserver将反序列化HTTP请求（解码），构造运行时对象（runtime object），并将它持久化到etcd。

### 横向扩展

kube-apiserver如何知道某一个资源的操作该如何处理呢？这在服务刚启动的时候会有非常复杂的配置步骤，让我们粗略看一下：

1. 当kube-apiserver启动时，会创建一个服务链(server chain)，允许apiserver进行聚合，这是提供多个apiserver的基础方式

```go
// cmd/kube-apiserver/app/server.go#L184
server, err := CreateServerChain(completeOptions, stopCh)
```

1. 作为默认实现的通用apiserver会被创建

```go
// cmd/kube-apiserver/app/server.go#L215
apiExtensionsServer, err := createAPIExtensionsServer(apiExtensionsConfig, genericapiserver.NewEmptyDelegate())
```

1. 生成的OpenAPI信息(schema)会填充到apiserver的配置中

```go
// cmd/kube-apiserver/app/server.go#L477
	genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(legacyscheme.Scheme, extensionsapiserver.Scheme, aggregatorscheme.Scheme))


```

1. kube-apiserver为每个API组配置一个存储服务提供器，它就是kube-apiserver访问和修改资源状态时的代理。

```go
// pkg/controlplane/instance.go#L591
for _, restStorageBuilder := range restStorageProviders {
		groupName := restStorageBuilder.GroupName()
		if !apiResourceConfigSource.AnyVersionForGroupEnabled(groupName) {
			klog.V(1).Infof("Skipping disabled API group %q.", groupName)
			continue
		}
		apiGroupInfo, enabled, err := restStorageBuilder.NewRESTStorage(apiResourceConfigSource, restOptionsGetter)
		if err != nil {
			return fmt.Errorf("problem initializing API group %q : %v", groupName, err)
		}
		if !enabled {
			klog.Warningf("API group %q is not enabled, skipping.", groupName)
			continue
        }
}
```

1. 为每一个不同版本的API组添加REST路由映射信息。这会运行kube-apiserver将请求映射到所匹配到的正确代理。

```go
// vendor/k8s.io/apiserver/pkg/server/genericapiserver.go#L439
r, err := apiGroupVersion.InstallREST(s.Handler.GoRestfulContainer)
复制代码
```

1. 在我们这个特定场景下，POST处理器会被注册，它将代理资源的创建操作

```go
// vendor/k8s.io/apiserver/pkg/endpoints/installer.go#816
case "POST": // Create a resource.
			var handler restful.RouteFunction
			if isNamedCreater {
				handler = restfulCreateNamedResource(namedCreater, reqScope, admit)
			} else {
				handler = restfulCreateResource(creater, reqScope, admit)
      }
```

小结一下，至此kube-apiserver完成了路由到内部资源操作代理的映射配置，当请求匹配后，就可以触发指定的操作代理了。

### pod存储流程

我们继续看pod创建的流程：

1. 基于注册的路由信息，当请求匹配到处理器链条中的某一个时，就会交由该处理器去处理。如果没有匹配的处理器，就返回给基于路径的处理器进行处理。但是如果没有注册路径处理器，则由notfound处理器返回404错误信息。
2. 幸运的是我们已经注册过createHandler了。它会做些什么呢？首先，它会解码HTTP请求体，并进行基础的验证，如提供的json是否符合相关版本API资源的要求
3. 进行审计和最终的准入检查
4. 通过存储代理将资源存储到etcd中。通常etcd的key是如下格式：/，它可以通过配置继续修改

```go
Create(ctx context.Context, name string, obj runtime.Object, createValidation ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error)
```

1. 检查是否有任何错误，没有错误时存储代理会通过get调用来确保资源对象确实被创建了。然后它会触发post-create handler和其他额外要求的装饰器。
2. 构造HTTP请求返回内容并发送

```go
// vendor/k8s.io/apiserver/pkg/endpoints/handlers/create.go#L49
func createHandler(r rest.NamedCreater, scope *RequestScope, admit admission.Interface, includeName bool) http.HandlerFunc {
		gv := scope.Kind.GroupVersion()
		s, err := negotiation.NegotiateInputSerializer(req, false, scope.Serializer)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		// 这个decoder构建比较关键，后边对body的解析就是通过它来完成的
		decoder := scope.Serializer.DecoderToVersion(s.Serializer, scope.HubGroupVersion)
		

		// 读取包体：序列化后的runtime
		body, err := limitedReadBody(req, scope.MaxRequestBodyBytes)
		if err != nil {
			scope.err(err, w, req)
			return
        }
		// 检查创建参数

		// 开始解码转换
		defaultGVK := scope.Kind
		original := r.New()
		trace.Step("About to convert to expected version")
		obj, gvk, err := decoder.Decode(body, &defaultGVK, original)
		// 省略
		trace.Step("Conversion done")

		// admission
		
		// 开始存储etcd
		trace.Step("About to store object in database")
		admissionAttributes := admission.NewAttributesRecord(obj, nil, scope.Kind, namespace, name, scope.Resource, scope.Subresource, admission.Create, options, dryrun.IsDryRun(options.DryRun), userInfo)
		requestFunc := func() (runtime.Object, error) {
			return r.Create(
				ctx,
				name,
				obj,
				rest.AdmissionToValidateObjectFunc(admit, admissionAttributes, scope),
				options,
			)
		}
		// 省略
		trace.Step("Object stored in database")

		// 构造HTTP返回结果
		code := http.StatusCreated
		status, ok := result.(*metav1.Status)
		if ok && err == nil && status.Code == 0 {
			status.Code = int32(code)
		}

		transformResponseObject(ctx, scope, trace, req, w, code, outputMediaType, result)
  // 至此，整个创建pod的HTTP请求就会返回，同时会返回创建后的对象
```

## 调度流程简介

调度器在控制面中是一个独立运行的模块，但是与其他控制器的运行方式完全相同：监听事件，然后尝试调谐状态。具体来说，调度器会过滤出所有在PodSpec中NodeName字段为空的pod，然后尝试为这些pod找到一个适合其运行的节点。

为了找到合适的节点，调度器将使用一个特有的调度算法。这个调度算法的工作方式如下两步：

1. 当调度器启动时，会注册一系列默认的预测器。这些预测器是很有效率的函数，当判断一个节点是否适合承载一个pod时，预测器会被执行。
2. 在挑选完合适的节点后，对这些节点会再执行一系列的优先级函数来对这些候选节点进行打分，以便进行适合度的排序。例如，为了尽可能的将工作负载分摊到整个集群中，调度器会更倾向于当前资源已分配更少的节点。当运行这些函数时，它会给每个节点打分，最终调度器会选择得分最高的节点。

当调度器将一个pod调度到一个节点后，那个节点上的kubelet就会接手开始具体的创建工作。

## 调度器实现介绍

### 核心流程

调度器的整个初始化流程此处从略，此处重点看一下通用调度器的调度执行的核心流程。

```go
// pkg/scheduler/scheduler.go#L311
// Run begins watching and scheduling. It starts scheduling and blocked until the context is done.
func (sched *Scheduler) Run(ctx context.Context) {
	sched.SchedulingQueue.Run()
	wait.UntilWithContext(ctx, sched.scheduleOne, 0)
	sched.SchedulingQueue.Close()
}
```

调度器开始运行时首先通过go程的方式运行调度队列，所有需要调度的pod都必须先放入该队列中，默认实现为优先队列。

核心调度入口为：scheduleOne，调度一个pod，UntilWithContext可以理解为一个死循环，直到外部告知退出时整个调度才会结束。通过函数注释可以知道，整个调度过程是顺序执行的。

```go
// pkg/scheduler/scheduler.go#L429
// scheduleOne does the entire scheduling workflow for a single pod. It is serialized on the scheduling algorithm's host fitting.
func (sched *Scheduler) scheduleOne(ctx context.Context) {
  // 先取出一个待调度的pod
  podInfo := sched.NextPod()
  // 一些列检查，如果失败则会直接退出
  
  // 按照算法进行调度
  scheduleResult, err := sched.Algorithm.Schedule(schedulingCycleCtx, fwk, state, pod)
  if err != nil{
    // 根据具体的错误进行相应处理，区分不可调度和调度失败的情况
  }
  
  // 设置pod的NodeName，通知cache调度成功的信息，同时可以保持继续调度而不用等待绑定成功
  // Tell the cache to assume that a pod now is running on a given node, even though it hasn't been bound yet.
	// This allows us to keep scheduling without waiting on binding to occur.
	assumedPodInfo := podInfo.DeepCopy()
	assumedPod := assumedPodInfo.Pod
	// assume modifies `assumedPod` by setting NodeName=scheduleResult.SuggestedHost
  	err = sched.assume(assumedPod, scheduleResult.SuggestedHost)
  
  	// 执行预留的插件的预留方法
  	// 执行"允许"插件
  runPermitStatus := fwk.RunPermitPlugins(schedulingCycleCtx, state, assumedPod, scheduleResult.SuggestedHost)
  // 异步执行节点绑定
  go func() {
    // 等待允许的状态为成功
    waitOnPermitStatus := fwk.WaitOnPermit(bindingCycleCtx, assumedPod)
    
    // 执行绑定前插件
    // Run "prebind" plugins.
	preBindStatus := fwk.RunPreBindPlugins(bindingCycleCtx, state, assumedPod, scheduleResult.SuggestedHost)
    
    // 执行绑定
    err := sched.bind(bindingCycleCtx, fwk, assumedPod, scheduleResult.SuggestedHost, state)
    // 执行绑定后插件
    fwk.RunPostBindPlugins(bindingCycleCtx, state, assumedPod, scheduleResult.SuggestedHost)
  }
}
```

### 调度算法

所有的调度算法都必须实现该接口，k8s提供了一个默认的通用型的调度算法。

```go
// pkg/scheduler/core/generic_scheduler.go#L95
// ScheduleAlgorithm is an interface implemented by things that know how to schedule pods
// onto machines.
// TODO: Rename this type.
type ScheduleAlgorithm interface {
	Schedule(context.Context, framework.Framework, *framework.CycleState, *v1.Pod) (scheduleResult ScheduleResult, err error)
	// Extenders returns a slice of extender config. This is exposed for
	// testing.
	Extenders() []framework.Extender
}
```

通用调度算法实现流程如下：

```go
// pkg/scheduler/core/generic_scheduler.go#L131
// Schedule tries to schedule the given pod to one of the nodes in the node list.
// If it succeeds, it will return the name of the node.
// If it fails, it will return a FitError error with reasons.
func (g *genericScheduler) Schedule(ctx context.Context, fwk framework.Framework, state *framework.CycleState, pod *v1.Pod) (result ScheduleResult, err error) {
  // 判断当前是否有可调度的节点
  if g.nodeInfoSnapshot.NumNodes() == 0{}
  
  // 找到所有合适的节点
  feasibleNodes, filteredNodesStatuses, err := g.findNodesThatFitPod(ctx, fwk, state, pod)
  
  // 候选节点打分排序
  priorityList, err := g.prioritizeNodes(ctx, fwk, state, pod, feasibleNodes)
  // 选择一个合适的节点
  host, err := g.selectHost(priorityList)
  // 返回调度结果
  return ScheduleResult{
		SuggestedHost:  host,
		EvaluatedNodes: len(feasibleNodes) + len(filteredNodesStatuses),
		FeasibleNodes:  len(feasibleNodes),
  }, err
}
```

另外预测器和打分器函数都是可以扩展的，可以通过--policy-config-fie参数项来自定义。这就增了一定程度的灵活性。管理员也可以通过独立的Deployment来运行自定义的调度器(本质是特殊逻辑的控制器)。如果PodSpec中schedulerName被设置了，Kubernetes无路如何都将会把这个pod的调度交给符合其所指定的名字的调度器。

### 节点绑定流程

当找到了一个合适的节点时，调度器就会创建一个Binding对象，其Name和UID可以匹配到该pod，其ObjectReference字段保存着所选择的节点的名称。这个对象将会通过POST请求发送给apiserver。

当apisever接收到该Binding对象，它会反序列化该对象，并且更新对应pod对象的下列字段：将NodeName设置为ObjectReference的值，添加相关的annotation，设置PodScheduled状态条件为True。

[kubernetes.io/docs/concep…](https://link.juejin.cn?target=https%3A%2F%2Fkubernetes.io%2Fdocs%2Fconcepts%2Fscheduling-eviction%2Fkube-scheduler%2F%23kube-scheduler-implementation)

# kubelet

kubelet是运行在k8s集群中每一个节点上的代理端，每个节点都会启动 kubelet进程，用来处理 Master 节点下发到本节点的任务，同时它也负责管理pod的生命周期以及其他的事情。kueblet实现了抽象的Kubernetes概念Pod到具体的构建模块、容器之间的转换逻辑。同时，它还负责处理所有这些与挂载卷、容器日志、垃圾回收，以及其他更重要事情相关的事务。

更多完整的介绍参考官方文档

[kubernetes.io/docs/refere…](https://link.juejin.cn?target=https%3A%2F%2Fkubernetes.io%2Fdocs%2Freference%2Fcommand-line-tools-reference%2Fkubelet%2F)

## 工作原理

![image-2](./image/image-2.awebp)

整体来看，kubelet启动了一个SyncLoop，所有工作都是围绕这个死循环展开的，功能十分庞杂，在这里我们重点关注一下kubelet创建pod的流程。

## 创建流程

### 整体介绍

当一个pod完成节点绑定后，就会触发kubelet的handler。kubelet监听到pod的变化，然后根据事件的类型进行不同的处理，有ADD、UPDATE、REMOVE、DELETE、等等，新增是ADD。

对所有新增的POD按照创建时间进行排序，保证最先创建的pod会最先被处理。然后逐个把pod加入到podManager中，podManager子模块负责管理这台机器上的pod信息、pod和mirrorPod之间的对应关系等等。所有被管理的pod都要出现在podManager中，如果没有，就认为这个pod被删除了。

如果操作类型是镜像pod，则执行镜像pod处理，后续操作跳过。

验证该pod是否可以在该节点运行，如果不可以直接拒绝，pod将会永久处于未就绪状态，不会自行恢复，需要人工干预。

通过dispatchWork把创建pod的工作下发给podWorkers子模块做异步处理。

在probeManager中添加pod，如果pod中定义了readiness和liveness健康检查，启动goroutine定期进行检测。

### 准备工作

在podworker具体创建容器前需要做一系列的准备工作（syncPod 注意大小写）此处巨复杂，先简单看看。

在这个方法中，主要完成以下几件事情：

- 如果是删除 pod，立即执行并返回
- 同步 podStatus 到 kubelet.statusManager
- 检查 pod 是否能运行在本节点，主要是权限检查（是否能使用主机网络模式，是否可以以 privileged 权限运行等）。如果没有权限，就删除本地旧的 pod 并返回错误信息
- 创建 containerManagar 对象，并且创建 pod level cgroup，更新 Qos level cgroup
- 如果是 static Pod，就创建或者更新对应的 mirrorPod
- 创建 pod 的数据目录，存放 volume 和 plugin 信息,如果定义了 pv，等待所有的 volume mount 完成（volumeManager 会在后台做这些事情）,如果有 image secrets，去 apiserver 获取对应的 secrets 数据
- 然后调用 kubelet.volumeManager 组件，等待它将 pod 所需要的所有外挂的 volume 都准备好。
- 调用 container runtime 的 SyncPod 方法，去实现真正的容器创建逻辑

这里所有的事情都和具体的容器没有关系，可以看到该方法是创建 pod 实体（即容器）之前需要完成的准备工作。

```go
// pkg/kubelet/kubelet.go#L1455
// syncPod is the transaction script for the sync of a single pod.
//
// This operation writes all events that are dispatched in order to provide
// the most accurate information possible about an error situation to aid debugging.
// Callers should not throw an event if this operation returns an error.
func (kl *Kubelet) syncPod(o syncPodOptions) error {
  // Call the container runtime's SyncPod callback
  result := kl.containerRuntime.SyncPod(pod, podStatus, pullSecrets, kl.backOff)
}
```

### 创建容器

containerRuntime子模块的SyncPod函数真正完成pod内容器的创建。

SyncPod 主要执行以下几个操作：

- 1、计算 sandbox 和 container 是否发生变化
- 2、创建 sandbox 容器
- 3、创建 init 容器
- 4、创建业务容器

这部分代码的注释非常完整，值得称赞！

```go
// pkg/kubelet/kuberuntime/kuberuntime_manager.go#L675
// SyncPod syncs the running pod into the desired pod by executing following steps:
//
//  1. Compute sandbox and container changes.
//  2. Kill pod sandbox if necessary.
//  3. Kill any containers that should not be running.
//  4. Create sandbox if necessary.
//  5. Create ephemeral containers.
//  6. Create init containers.
//  7. Create normal containers.
func (m *kubeGenericRuntimeManager) SyncPod(pod *v1.Pod, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, backOff *flowcontrol.Backoff) (result kubecontainer.PodSyncResult) {
  
  start := func(typeName string, spec *startSpec) error {
    if msg, err := m.startContainer(podSandboxID, podSandboxConfig, spec, pod, podStatus, pullSecrets, podIP, podIPs);
  }
  // Step 6: start the init container.
  if err := start("init container", containerStartSpec(container));
  // Step 7: start containers in podContainerChanges.ContainersToStart.
  for _, idx := range podContainerChanges.ContainersToStart {
		start("container", containerStartSpec(&pod.Spec.Containers[idx]))
  }
}
```

### 启动容器

最终由startContainer完成容器的启动

主要有以下步骤：

- 1、拉取镜像
- 2、生成业务容器的配置信息
- 3、调用运行时服务 api 创建容器，注意在v1.20版本中开始标记弃用dockershim，在v1.23中将彻底移除，在此之前需要提供受支持的容器运行时，更多内容可以参考文档xxxx
- 4、启动容器
- 5、执行 post start hook

```go
// pkg/kubelet/kuberuntime/kuberuntime_container.go#L134
// startContainer starts a container and returns a message indicates why it is failed on error.
// It starts the container through the following steps:
// * pull the image
// * create the container
// * start the container
// * run the post start lifecycle hooks (if applicable)
func (m *kubeGenericRuntimeManager) startContainer(podSandboxID string, podSandboxConfig *runtimeapi.PodSandboxConfig, spec *startSpec, pod *v1.Pod, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, podIP string, podIPs []string) (string, error) {
}
```

## 小结

通过pod的创建流程可以看到kubelet承担了庞大的基础管理和操作任务，进入到kubelet内部后也会发现，kubelet的整体架构也体现了它的复杂性。pod的创建只是其很小的一部分工作。最后附一张kubelet整体模块架构图。

![image-3](./image/image-3.awebp)

# 


